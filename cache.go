package mermaid

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CachedResult represents a serialized render result stored in cache.
type CachedResult struct {
	Mode      RenderMode       `json:"mode"`
	Protocol  GraphicsProtocol `json:"protocol"`
	ImageData []byte           `json:"image_data,omitempty"`
	Output    string           `json:"output"`
	CreatedAt time.Time        `json:"created_at"`
}

// Cache defines the interface for content-addressed diagram caching.
type Cache interface {
	Get(key string) (*CachedResult, bool)
	Set(key string, res *CachedResult, ttl time.Duration) error
	Clear() error
}

// ComputeCacheKey computes a SHA-256 content-addressed hash based on diagram source and rendering configuration.
func ComputeCacheKey(source string, cfg *Config) string {
	h := sha256.New()
	h.Write([]byte(source))
	h.Write([]byte(fmt.Sprintf(":%d:%d:%s:%.2f:%s:%s:%d:%t:%t:%t:%s",
		cfg.Mode,
		cfg.GraphicsProtocol,
		cfg.Theme,
		cfg.Scale,
		cfg.Width,
		cfg.Height,
		cfg.Columns,
		cfg.SharpEdges,
		cfg.BoxFrame,
		cfg.PreserveAspectRatio,
		cfg.Title,
	)))
	return hex.EncodeToString(h.Sum(nil))
}

// MemoryCache is an in-memory thread-safe cache for diagrams.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

type memoryItem struct {
	result    CachedResult
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory diagram cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]memoryItem),
	}
}

// Get retrieves an item from the memory cache if present and not expired.
func (c *MemoryCache) Get(key string) (*CachedResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return nil, false
	}
	resCopy := item.result
	return &resCopy, true
}

// Set stores an item in the memory cache.
func (c *MemoryCache) Set(key string, res *CachedResult, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = memoryItem{
		result:    *res,
		expiresAt: expiresAt,
	}
	return nil
}

// Clear clears all items from the memory cache.
func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]memoryItem)
	return nil
}

// DiskCache implements a persistent disk cache stored in a filesystem directory with an in-memory tier.
type DiskCache struct {
	dir string
	mem *MemoryCache
	mu  sync.RWMutex
}

// DefaultCacheDir returns the default cache directory for the user.
func DefaultCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "golang-mermaid")
}

// NewDiskCache creates a disk cache storing cached items in dir.
func NewDiskCache(dir string) (*DiskCache, error) {
	if dir == "" {
		dir = DefaultCacheDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory %q: %w", dir, err)
	}
	return &DiskCache{
		dir: dir,
		mem: NewMemoryCache(),
	}, nil
}

// Dir returns the directory path for the disk cache.
func (d *DiskCache) Dir() string {
	return d.dir
}

// Get retrieves a cached diagram from memory tier or disk.
func (d *DiskCache) Get(key string) (*CachedResult, bool) {
	// Check memory cache tier first
	if res, ok := d.mem.Get(key); ok {
		return res, true
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	filePath := filepath.Join(d.dir, key+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	var stored struct {
		Result    CachedResult `json:"result"`
		ExpiresAt time.Time    `json:"expires_at"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, false
	}

	if !stored.ExpiresAt.IsZero() && time.Now().After(stored.ExpiresAt) {
		_ = os.Remove(filePath)
		return nil, false
	}

	// Populate memory tier
	var remainingTTL time.Duration
	if !stored.ExpiresAt.IsZero() {
		remainingTTL = time.Until(stored.ExpiresAt)
	}
	_ = d.mem.Set(key, &stored.Result, remainingTTL)

	return &stored.Result, true
}

// Set writes a cached diagram to memory tier and disk atomically.
func (d *DiskCache) Set(key string, res *CachedResult, ttl time.Duration) error {
	_ = d.mem.Set(key, res, ttl)

	d.mu.Lock()
	defer d.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	stored := struct {
		Result    CachedResult `json:"result"`
		ExpiresAt time.Time    `json:"expires_at"`
	}{
		Result:    *res,
		ExpiresAt: expiresAt,
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal cached result: %w", err)
	}

	filePath := filepath.Join(d.dir, key+".json")
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}
	return os.Rename(tmpPath, filePath)
}

// Clear removes all cached items from memory tier and disk.
func (d *DiskCache) Clear() error {
	_ = d.mem.Clear()

	d.mu.Lock()
	defer d.mu.Unlock()

	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			_ = os.Remove(filepath.Join(d.dir, entry.Name()))
		}
	}
	return nil
}

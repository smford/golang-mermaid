package mermaid

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestComputeCacheKey(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()
	cfg2.Theme = "slate"

	key1 := ComputeCacheKey("graph TD\n A-->B", &cfg1)
	key1Repeat := ComputeCacheKey("graph TD\n A-->B", &cfg1)
	key2 := ComputeCacheKey("graph TD\n A-->B", &cfg2)
	key3 := ComputeCacheKey("graph TD\n A-->C", &cfg1)

	if key1 != key1Repeat {
		t.Errorf("expected identical keys for same source and config")
	}
	if key1 == key2 {
		t.Errorf("expected different keys when theme changes")
	}
	if key1 == key3 {
		t.Errorf("expected different keys when source changes")
	}
}

func TestMemoryCache(t *testing.T) {
	c := NewMemoryCache()

	item := &CachedResult{
		Mode:      ModeUnicode,
		Output:    "diagram-output",
		CreatedAt: time.Now(),
	}

	// 1. Get non-existent
	if _, ok := c.Get("missing"); ok {
		t.Error("expected false for missing key")
	}

	// 2. Set with TTL
	if err := c.Set("k1", item, 50*time.Millisecond); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 3. Get existing
	got, ok := c.Get("k1")
	if !ok || got.Output != "diagram-output" {
		t.Errorf("Get failed, got: %v", got)
	}

	// 4. Wait for TTL expiration
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get("k1"); ok {
		t.Error("expected expired key to return false")
	}

	// 5. Clear
	_ = c.Set("k2", item, 0)
	if err := c.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if _, ok := c.Get("k2"); ok {
		t.Error("expected key to be cleared")
	}
}

func TestDiskCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mermaid-cache-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dc, err := NewDiskCache(tmpDir)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	item := &CachedResult{
		Mode:      ModeImage,
		Protocol:  ProtocolKitty,
		ImageData: []byte("PNG_BYTES"),
		Output:    "\033_Gtest\033\\",
		CreatedAt: time.Now(),
	}

	// Set and Get
	if err := dc.Set("test-key", item, 1*time.Hour); err != nil {
		t.Fatalf("dc.Set failed: %v", err)
	}

	// Verify file was created on disk
	expectedFile := filepath.Join(tmpDir, "test-key.json")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("expected cache file on disk: %v", err)
	}

	// Read back through fresh DiskCache instance
	dc2, err := NewDiskCache(tmpDir)
	if err != nil {
		t.Fatalf("NewDiskCache 2 failed: %v", err)
	}

	got, ok := dc2.Get("test-key")
	if !ok {
		t.Fatal("expected test-key to be found in disk cache")
	}
	if got.Protocol != ProtocolKitty || string(got.ImageData) != "PNG_BYTES" {
		t.Errorf("retrieved item does not match: %+v", got)
	}

	// Test Clear
	if err := dc2.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if _, ok := dc2.Get("test-key"); ok {
		t.Error("expected key to be cleared from disk")
	}
}

func TestPrinter_CacheIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "printer-cache-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	callCount := 0
	mockImg := &countingImageRenderer{data: []byte("CACHED_PNG"), counter: &callCount}

	printer := New(
		WithMode(ModeImage),
		WithGraphicsProtocol(ProtocolKitty),
		WithImageRenderer(mockImg),
		WithCacheDir(tmpDir),
		WithCache(true),
	)

	diagram := "graph TD\n NodeA --> NodeB"

	// First render: cache miss
	res1, err := printer.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("render 1 failed: %v", err)
	}
	if res1.CacheHit {
		t.Error("expected CacheHit=false on first render")
	}
	if callCount != 1 {
		t.Errorf("expected 1 image render call, got %d", callCount)
	}

	// Second render: cache hit
	res2, err := printer.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("render 2 failed: %v", err)
	}
	if !res2.CacheHit {
		t.Error("expected CacheHit=true on second render")
	}
	if callCount != 1 {
		t.Errorf("expected still 1 image render call after cache hit, got %d", callCount)
	}
	if res2.Output != res1.Output {
		t.Errorf("cached output differs from original output")
	}
}

func TestCache_Concurrency(t *testing.T) {
	c := NewMemoryCache()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "concurrent-key"
			_ = c.Set(key, &CachedResult{Output: "val"}, 100*time.Millisecond)
			_, _ = c.Get(key)
		}(i)
	}
	wg.Wait()
}

type countingImageRenderer struct {
	data    []byte
	counter *int
}

func (m *countingImageRenderer) RenderImage(ctx context.Context, mermaidSource string) ([]byte, error) {
	*m.counter++
	return m.data, nil
}

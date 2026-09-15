package mermaid

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Common errors for image rendering.
var (
	ErrEmptySource    = errors.New("mermaid source is empty")
	ErrCLINotFound    = errors.New("mermaid-cli (mmdc) binary not found on PATH")
	ErrAllRenderers   = errors.New("all image renderers failed")
	ErrRenderTimeout  = errors.New("image rendering timed out")
	ErrInvalidStatus  = errors.New("unexpected HTTP response status")
	ErrOfflineNoCLI   = errors.New("offline mode active: local mmdc CLI not found, remote renderers disabled")
)

// ImageRenderer is the interface for converting Mermaid diagram syntax to image bytes (typically PNG).
type ImageRenderer interface {
	RenderImage(ctx context.Context, mermaidSource string) ([]byte, error)
}

// KrokiRenderer renders Mermaid diagrams using the Kroki REST API.
type KrokiRenderer struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	Headers    map[string]string
}

// NewKrokiRenderer returns a new KrokiRenderer with the given base URL.
// If baseURL is empty, "https://kroki.io" is used.
func NewKrokiRenderer(baseURL string, timeout time.Duration) *KrokiRenderer {
	if baseURL == "" {
		baseURL = "https://kroki.io"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &KrokiRenderer{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		UserAgent: "golang-mermaid/1.0",
		Headers:   make(map[string]string),
	}
}

// RenderImage POSTs the Mermaid diagram to Kroki and returns the PNG bytes.
func (k *KrokiRenderer) RenderImage(ctx context.Context, mermaidSource string) ([]byte, error) {
	mermaidSource = strings.TrimSpace(mermaidSource)
	if mermaidSource == "" {
		return nil, ErrEmptySource
	}

	endpoint := fmt.Sprintf("%s/mermaid/png", k.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(mermaidSource))
	if err != nil {
		return nil, fmt.Errorf("kroki: create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Accept", "image/png, */*")
	if k.UserAgent != "" {
		req.Header.Set("User-Agent", k.UserAgent)
	}
	for key, val := range k.Headers {
		req.Header.Set(key, val)
	}

	resp, err := k.HTTPClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
			return nil, fmt.Errorf("kroki: %w: %v", ErrRenderTimeout, ctx.Err())
		}
		return nil, fmt.Errorf("kroki request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kroki: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return nil, fmt.Errorf("kroki error (HTTP %d): %s", resp.StatusCode, msg)
	}

	return body, nil
}

// MermaidInkRenderer renders Mermaid diagrams using the mermaid.ink public service.
type MermaidInkRenderer struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewMermaidInkRenderer returns a new MermaidInkRenderer.
func NewMermaidInkRenderer(timeout time.Duration) *MermaidInkRenderer {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &MermaidInkRenderer{
		BaseURL: "https://mermaid.ink",
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		UserAgent: "golang-mermaid/1.0",
	}
}

// RenderImage sends the diagram to mermaid.ink and returns the image bytes.
func (m *MermaidInkRenderer) RenderImage(ctx context.Context, mermaidSource string) ([]byte, error) {
	mermaidSource = strings.TrimSpace(mermaidSource)
	if mermaidSource == "" {
		return nil, ErrEmptySource
	}

	state := map[string]string{"code": mermaidSource}
	jsonBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("mermaid.ink: json encode: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(jsonBytes)
	endpoint := fmt.Sprintf("%s/img/%s", strings.TrimSuffix(m.BaseURL, "/"), encoded)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("mermaid.ink: create request: %w", err)
	}
	if m.UserAgent != "" {
		req.Header.Set("User-Agent", m.UserAgent)
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mermaid.ink request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mermaid.ink: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return nil, fmt.Errorf("mermaid.ink error (HTTP %d): %s", resp.StatusCode, msg)
	}

	return body, nil
}

// LocalCLIRenderer renders Mermaid diagrams using the local `mmdc` CLI (mermaid-cli).
type LocalCLIRenderer struct {
	BinaryPath string
	Theme      string
	Scale      float64
}

// NewLocalCLIRenderer returns a LocalCLIRenderer using the specified mmdc binary path.
// If binaryPath is empty, it searches for "mmdc" on PATH.
func NewLocalCLIRenderer(binaryPath, theme string) (*LocalCLIRenderer, error) {
	if binaryPath == "" {
		p, err := exec.LookPath("mmdc")
		if err != nil {
			return nil, ErrCLINotFound
		}
		binaryPath = p
	}
	return &LocalCLIRenderer{
		BinaryPath: binaryPath,
		Theme:      theme,
		Scale:      1.0,
	}, nil
}

// RenderImage executes mmdc with temporary input and output files.
func (c *LocalCLIRenderer) RenderImage(ctx context.Context, mermaidSource string) ([]byte, error) {
	mermaidSource = strings.TrimSpace(mermaidSource)
	if mermaidSource == "" {
		return nil, ErrEmptySource
	}

	tmpDir, err := os.MkdirTemp("", "golang-mermaid-*")
	if err != nil {
		return nil, fmt.Errorf("local cli: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputFile := filepath.Join(tmpDir, "diagram.mmd")
	outputFile := filepath.Join(tmpDir, "output.png")

	if err := os.WriteFile(inputFile, []byte(mermaidSource), 0600); err != nil {
		return nil, fmt.Errorf("local cli: write input file: %w", err)
	}

	args := []string{"-i", inputFile, "-o", outputFile}
	if c.Theme != "" && c.Theme != "default" {
		args = append(args, "-t", c.Theme)
	}
	if c.Scale > 0 && c.Scale != 1.0 {
		args = append(args, "-s", fmt.Sprintf("%g", c.Scale))
	}

	cmd := exec.CommandContext(ctx, c.BinaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("local cli mmdc failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("local cli: read output file: %w", err)
	}

	return data, nil
}

// ResilientImageRenderer chains multiple ImageRenderers in order.
// If the first renderer fails or times out, it tries the subsequent ones.
type ResilientImageRenderer struct {
	Renderers []ImageRenderer
}

// NewResilientImageRenderer creates a composite renderer with sensible production defaults:
// 1. Local CLI mmdc (if installed)
// 2. Kroki HTTP API (fast, reliable diagram service)
// 3. Mermaid.ink HTTP API (secondary fallback)
func NewResilientImageRenderer(krokiBaseURL string, timeout time.Duration) *ResilientImageRenderer {
	return NewResilientImageRendererWithScale(krokiBaseURL, timeout, 1.0)
}

// NewResilientImageRendererWithScale creates a composite renderer with a specific rasterization scale factor.
func NewResilientImageRendererWithScale(krokiBaseURL string, timeout time.Duration, scale float64) *ResilientImageRenderer {
	var renderers []ImageRenderer

	// Try local mmdc first if installed on the system
	if cli, err := NewLocalCLIRenderer("", "default"); err == nil {
		if scale > 0 {
			cli.Scale = scale
		}
		renderers = append(renderers, cli)
	}

	// Always add Kroki
	renderers = append(renderers, NewKrokiRenderer(krokiBaseURL, timeout))

	// Add Mermaid.ink as a secondary remote fallback
	renderers = append(renderers, NewMermaidInkRenderer(timeout))

	return &ResilientImageRenderer{
		Renderers: renderers,
	}
}

// NewOfflineImageRenderer creates an ImageRenderer that strictly avoids all network calls,
// attempting only the local mmdc binary if installed.
func NewOfflineImageRenderer(scale float64) *ResilientImageRenderer {
	var renderers []ImageRenderer
	if cli, err := NewLocalCLIRenderer("", "default"); err == nil {
		if scale > 0 {
			cli.Scale = scale
		}
		renderers = append(renderers, cli)
	}
	return &ResilientImageRenderer{
		Renderers: renderers,
	}
}

// RenderImage attempts each configured renderer until one succeeds.
func (r *ResilientImageRenderer) RenderImage(ctx context.Context, mermaidSource string) ([]byte, error) {
	if len(r.Renderers) == 0 {
		return nil, ErrOfflineNoCLI
	}

	var errs []string
	numRenderers := len(r.Renderers)

	for i, renderer := range r.Renderers {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Calculate per-renderer sub-timeout to prevent one slow renderer from starving the chain
		subCtx := ctx
		var cancel context.CancelFunc
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return nil, ctx.Err()
			}
			renderersLeft := time.Duration(numRenderers - i)
			budget := remaining / renderersLeft
			// Ensure at least 3 seconds if overall budget allows
			if budget < 3*time.Second && remaining >= 3*time.Second {
				budget = 3 * time.Second
			} else if budget > remaining {
				budget = remaining
			}
			subCtx, cancel = context.WithTimeout(ctx, budget)
		}

		data, err := renderer.RenderImage(subCtx, mermaidSource)
		if cancel != nil {
			cancel()
		}

		if err == nil && len(data) > 0 {
			return data, nil
		}
		errs = append(errs, fmt.Sprintf("renderer #%d failed: %v", i+1, err))
	}

	return nil, fmt.Errorf("%w: %s", ErrAllRenderers, strings.Join(errs, "; "))
}

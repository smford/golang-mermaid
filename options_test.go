package mermaid

import (
	"bytes"
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	buf := new(bytes.Buffer)
	fallbackTriggered := false

	cfg := DefaultConfig()
	opts := []Option{
		WithMode(ModeASCII),
		WithWriter(buf),
		WithWidth("600px"),
		WithHeight("300px"),
		WithPreserveAspectRatio(false),
		WithAllowCompatibleTerminals(false),
		WithDisableFallback(true),
		WithForceTTY(true),
		WithTimeout(5 * time.Second),
		WithTheme("forest"),
		WithColumns(130),
		WithPadding(2, 1),
		WithSharpEdges(true),
		WithHyperlinks(true),
		WithBoxFrame(true),
		WithTitle("Sample System"),
		WithScale(2.5),
		WithGraphicsProtocol(ProtocolKitty),
		WithCache(true),
		WithCacheDir("/tmp/test-cache"),
		WithCacheTTL(10 * time.Minute),
		WithOffline(true),
		WithTerminalProbe(true),
		WithOnFallback(func(reason string, err error) {
			fallbackTriggered = true
		}),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Mode != ModeASCII {
		t.Errorf("expected ModeASCII, got %v", cfg.Mode)
	}
	if cfg.Writer != buf {
		t.Errorf("expected custom writer")
	}
	if cfg.Width != "600px" {
		t.Errorf("expected width 600px, got %s", cfg.Width)
	}
	if cfg.Height != "300px" {
		t.Errorf("expected height 300px, got %s", cfg.Height)
	}
	if cfg.PreserveAspectRatio != false {
		t.Errorf("expected preserveAspectRatio false")
	}
	if cfg.AllowCompatibleTerminals != false {
		t.Errorf("expected allowCompatibleTerminals false")
	}
	if cfg.DisableFallback != true {
		t.Errorf("expected disableFallback true")
	}
	if cfg.ForceTTY != true {
		t.Errorf("expected forceTTY true")
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", cfg.Timeout)
	}
	if cfg.Theme != "forest" {
		t.Errorf("expected theme forest, got %s", cfg.Theme)
	}
	if cfg.Columns != 130 {
		t.Errorf("expected columns 130, got %d", cfg.Columns)
	}
	if cfg.PaddingX != 2 || cfg.PaddingY != 1 {
		t.Errorf("expected padding (2, 1), got (%d, %d)", cfg.PaddingX, cfg.PaddingY)
	}
	if !cfg.SharpEdges {
		t.Errorf("expected sharpEdges true")
	}
	if !cfg.Hyperlinks {
		t.Errorf("expected hyperlinks true")
	}
	if !cfg.BoxFrame {
		t.Errorf("expected boxFrame true")
	}
	if cfg.Title != "Sample System" {
		t.Errorf("expected title 'Sample System', got %q", cfg.Title)
	}
	if cfg.Scale != 2.5 {
		t.Errorf("expected scale 2.5, got %v", cfg.Scale)
	}
	if cfg.GraphicsProtocol != ProtocolKitty {
		t.Errorf("expected graphics protocol ProtocolKitty, got %v", cfg.GraphicsProtocol)
	}
	if !cfg.CacheEnabled {
		t.Errorf("expected CacheEnabled true")
	}
	if cfg.CacheDir != "/tmp/test-cache" {
		t.Errorf("expected CacheDir /tmp/test-cache, got %s", cfg.CacheDir)
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Errorf("expected CacheTTL 10m, got %v", cfg.CacheTTL)
	}
	if !cfg.Offline {
		t.Errorf("expected Offline true")
	}
	if !cfg.TerminalProbe {
		t.Errorf("expected TerminalProbe true")
	}
	if cfg.OnFallback != nil {
		cfg.OnFallback("test", nil)
		if !fallbackTriggered {
			t.Errorf("expected fallback hook to be called")
		}
	} else {
		t.Errorf("expected OnFallback to be set")
	}
}

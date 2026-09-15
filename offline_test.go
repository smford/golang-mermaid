package mermaid

import (
	"context"
	"strings"
	"testing"
)

func TestOfflineImageRenderer_NoCLI(t *testing.T) {
	// Create an offline renderer with no CLI installed (or non-existent path)
	renderer := &ResilientImageRenderer{Renderers: nil}
	_, err := renderer.RenderImage(context.Background(), "graph TD\n A-->B")
	if err != ErrOfflineNoCLI {
		t.Errorf("expected ErrOfflineNoCLI, got: %v", err)
	}
}

func TestPrinter_OfflineMode_AutoFallback(t *testing.T) {
	var fallbackReasonRecorded string
	fallbackHookCalled := false

	itermEnv := mapEnv{
		"TERM_PROGRAM": "iTerm.app",
	}

	p := New(
		WithMode(ModeAuto),
		WithTerminalEnv(itermEnv),
		WithForceTTY(true),
		WithOffline(true),
		// Empty renderer to simulate environment without local mmdc CLI
		WithImageRenderer(&ResilientImageRenderer{Renderers: nil}),
		WithOnFallback(func(reason string, err error) {
			fallbackHookCalled = true
			fallbackReasonRecorded = reason
		}),
	)

	diagram := "graph TD\n A[AirGappedHost] --> B[InternalDB]"
	res, err := p.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if !res.FallbackOccurred {
		t.Error("expected FallbackOccurred=true when offline mode cannot render image")
	}
	if !fallbackHookCalled {
		t.Error("expected fallback hook to be called in offline mode")
	}
	if !strings.Contains(fallbackReasonRecorded, "offline mode") {
		t.Errorf("expected fallback reason to mention offline mode, got: %q", fallbackReasonRecorded)
	}
	if res.Mode != ModeUnicode {
		t.Errorf("expected ModeUnicode, got: %v", res.Mode)
	}
	if !strings.Contains(res.Output, "AirGappedHost") {
		t.Errorf("expected diagram output to contain node text, got:\n%s", res.Output)
	}
}

func TestPrinter_OfflineMode_DisableFallback(t *testing.T) {
	p := New(
		WithMode(ModeImage),
		WithOffline(true),
		WithDisableFallback(true),
		WithImageRenderer(&ResilientImageRenderer{Renderers: nil}),
	)

	_, err := p.Render(context.Background(), "graph TD\n A-->B")
	if err == nil {
		t.Fatal("expected error in ModeImage with DisableFallback in offline mode, got nil")
	}
}

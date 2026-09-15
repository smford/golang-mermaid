package mermaid

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrinter_AutoFallback_NonITerm(t *testing.T) {
	buf := new(bytes.Buffer)
	fallbackInvoked := false
	var fallbackReasonRecorded string

	nonITermEnv := mapEnv{
		"TERM_PROGRAM": "Apple_Terminal",
	}

	p := New(
		WithWriter(buf),
		WithMode(ModeAuto),
		WithTerminalEnv(nonITermEnv),
		WithForceTTY(true),
		WithOnFallback(func(reason string, err error) {
			fallbackInvoked = true
			fallbackReasonRecorded = reason
		}),
	)

	diagram := "graph TD\n    A[LoadBalancer] --> B[AppServer]"
	res, err := p.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.FallbackOccurred {
		t.Error("expected FallbackOccurred=true for non-iTerm terminal")
	}
	if !fallbackInvoked {
		t.Error("expected OnFallback hook to be called")
	}
	if !strings.Contains(fallbackReasonRecorded, "does not support iTerm2") {
		t.Errorf("unexpected fallback reason: %q", fallbackReasonRecorded)
	}
	if res.Mode != ModeUnicode {
		t.Errorf("expected ModeUnicode, got: %v", res.Mode)
	}
	if !strings.Contains(res.Output, "LoadBalancer") {
		t.Errorf("expected diagram output to contain LoadBalancer, got:\n%s", res.Output)
	}
}

func TestPrinter_Auto_ITermSuccess(t *testing.T) {
	buf := new(bytes.Buffer)
	mockImg := &mockImageRenderer{data: []byte("TEST_PNG_DATA")}

	itermEnv := mapEnv{
		"TERM_PROGRAM": "iTerm.app",
	}

	p := New(
		WithWriter(buf),
		WithMode(ModeAuto),
		WithTerminalEnv(itermEnv),
		WithForceTTY(true),
		WithImageRenderer(mockImg),
	)

	diagram := "graph TD\n    A --> B"
	res, err := p.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.FallbackOccurred {
		t.Error("did not expect fallback when iTerm is present and image succeeds")
	}
	if res.Mode != ModeImage {
		t.Errorf("expected ModeImage, got: %v", res.Mode)
	}
	if !strings.HasPrefix(res.Output, "\033]1337;File=") {
		t.Errorf("expected iTerm OSC 1337 sequence, got: %q", res.Output)
	}
}

func TestPrinter_Auto_ImageFailureFallback(t *testing.T) {
	buf := new(bytes.Buffer)
	fallbackInvoked := false
	mockFailImg := &mockImageRenderer{err: errors.New("upstream timeout")}

	itermEnv := mapEnv{
		"TERM_PROGRAM": "iTerm.app",
	}

	p := New(
		WithWriter(buf),
		WithMode(ModeAuto),
		WithTerminalEnv(itermEnv),
		WithForceTTY(true),
		WithImageRenderer(mockFailImg),
		WithOnFallback(func(reason string, err error) {
			fallbackInvoked = true
		}),
	)

	diagram := "graph TD\n    A[Primary] --> B[Secondary]"
	res, err := p.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.FallbackOccurred {
		t.Error("expected FallbackOccurred=true when image rendering fails")
	}
	if !fallbackInvoked {
		t.Error("expected OnFallback hook to be called on failure")
	}
	if res.Mode != ModeUnicode {
		t.Errorf("expected ModeUnicode, got: %v", res.Mode)
	}
	if !strings.Contains(res.Output, "Primary") {
		t.Errorf("expected fallback text to contain 'Primary', got:\n%s", res.Output)
	}
}

func TestPrinter_ModeImage_DisableFallback(t *testing.T) {
	mockFailImg := &mockImageRenderer{err: errors.New("cannot reach kroki")}

	p := New(
		WithMode(ModeImage),
		WithImageRenderer(mockFailImg),
		WithDisableFallback(true),
	)

	_, err := p.Render(context.Background(), "graph TD; A-->B;")
	if err == nil {
		t.Fatal("expected error when DisableFallback is true and image fails, got nil")
	}
}

func TestPrinter_ModeASCII(t *testing.T) {
	buf := new(bytes.Buffer)

	p := New(
		WithWriter(buf),
		WithMode(ModeASCII),
	)

	diagram := "graph TD\n    A[Client] --> B[Database]"
	err := p.Print(diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Client") || !strings.Contains(output, "Database") {
		t.Errorf("expected diagram output, got:\n%s", output)
	}
	if strings.Contains(output, "┌") {
		t.Errorf("expected strict ASCII, found unicode box character '┌'")
	}
}

func TestPrinter_PrintFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.mmd")
	content := "graph TD\n    A[ServiceA] --> B[ServiceB]\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	buf := new(bytes.Buffer)
	p := New(
		WithWriter(buf),
		WithMode(ModeASCII),
	)

	err := p.PrintFile(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "ServiceA") {
		t.Errorf("expected ServiceA in output, got: %s", buf.String())
	}
}

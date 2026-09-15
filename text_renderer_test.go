package mermaid

import (
	"context"
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestMmaidTextRenderer_Flowchart(t *testing.T) {
	renderer := NewMmaidTextRenderer(false, "default")
	diagram := "graph TD\n    A[Client] --> B[Server]"

	output, err := renderer.RenderText(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Client") || !strings.Contains(output, "Server") {
		t.Errorf("expected Client and Server in output, got:\n%s", output)
	}
}

func TestMmaidTextRenderer_StrictASCII(t *testing.T) {
	renderer := NewMmaidTextRenderer(true, "default")
	diagram := "graph TD\n    A[Client] --> B[Server]"

	output, err := renderer.RenderText(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Client") || !strings.Contains(output, "Server") {
		t.Errorf("expected Client and Server in output, got:\n%s", output)
	}
	// Strict ASCII uses '+' or '-' or '|' instead of unicode box characters like '┌'
	if strings.Contains(output, "┌") {
		t.Errorf("expected no Unicode box characters in strict ASCII mode")
	}
}

func TestImageToASCII(t *testing.T) {
	// Create a simple 10x10 test image with black square on white background
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if x >= 3 && x <= 7 && y >= 3 && y <= 7 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}

	ascii := ImageToASCII(img, 20)
	if len(ascii) == 0 {
		t.Fatal("expected non-empty ASCII output")
	}
}

func TestFallbackTextRenderer_RawFallback(t *testing.T) {
	// Fallback renderer when primary fails and secondary is nil
	fallback := &FallbackTextRenderer{
		PrimaryRenderer:   nil,
		SecondaryRenderer: nil,
	}

	raw := "stateDiagram-v2\n    [*] --> Active"
	out, err := fallback.RenderText(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Active") {
		t.Errorf("expected raw diagram in fallback output, got:\n%s", out)
	}
}

func TestVisibleRuneWidth(t *testing.T) {
	plain := "Hello World"
	if w := VisibleRuneWidth(plain); w != 11 {
		t.Errorf("expected 11, got %d", w)
	}

	withColor := "\033[36m┌──────────┐\033[0m"
	if w := VisibleRuneWidth(withColor); w != 12 {
		t.Errorf("expected visible width 12, got %d", w)
	}
}

func TestPolishTextDiagram(t *testing.T) {
	raw := "\n\n  Line 1   \n\n\n\n  Line 2   \n\n\n"
	polished := PolishTextDiagram(raw)

	if strings.HasPrefix(polished, "\n") {
		t.Error("expected leading blank lines to be stripped")
	}
	if strings.Contains(polished, "   \n") {
		t.Error("expected trailing spaces on lines to be trimmed")
	}
	if strings.Contains(polished, "\n\n\n") {
		t.Error("expected multiple blank lines to be collapsed to at most one")
	}
	if !strings.Contains(polished, "Line 1") || !strings.Contains(polished, "Line 2") {
		t.Error("expected lines to be preserved")
	}
}

func TestWrapInBoxFrame(t *testing.T) {
	content := "┌──────┐\n│ Node │\n└──────┘"
	framed := WrapInBoxFrame(content, "Test System", false, false, false)

	if !strings.Contains(framed, "Test System") {
		t.Errorf("expected title in framed output, got:\n%s", framed)
	}
	if !strings.Contains(framed, "╭") || !strings.Contains(framed, "╯") {
		t.Errorf("expected rounded corner glyphs in framed output, got:\n%s", framed)
	}

	// Test strict ASCII frame
	asciiFramed := WrapInBoxFrame(content, "ASCII System", false, true, false)
	if !strings.Contains(asciiFramed, "+") {
		t.Errorf("expected + corner glyph in strict ASCII frame, got:\n%s", asciiFramed)
	}
}

func TestMmaidTextRenderer_ThemesAndFrame(t *testing.T) {
	renderer := &MmaidTextRenderer{
		StrictASCII: false,
		Theme:       "slate",
		BoxFrame:    true,
		Title:       "Architecture",
		Columns:     120,
	}

	diagram := "graph TD\n    A[Client] --> B[Server]"
	out, err := renderer.RenderText(context.Background(), diagram)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Architecture") {
		t.Errorf("expected title in output")
	}
	if !strings.Contains(out, "Client") || !strings.Contains(out, "Server") {
		t.Errorf("expected diagram nodes in output")
	}
}


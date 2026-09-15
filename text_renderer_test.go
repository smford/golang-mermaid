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

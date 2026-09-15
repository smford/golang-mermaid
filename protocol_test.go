package mermaid

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestGraphicsProtocol_String(t *testing.T) {
	tests := []struct {
		proto    GraphicsProtocol
		expected string
	}{
		{ProtocolAuto, "auto"},
		{ProtocolITerm2, "iterm2"},
		{ProtocolKitty, "kitty"},
		{ProtocolSixel, "sixel"},
		{ProtocolNone, "none"},
		{GraphicsProtocol(999), "unknown"},
	}

	for _, tc := range tests {
		if tc.proto.String() != tc.expected {
			t.Errorf("proto.String() = %q, expected %q", tc.proto.String(), tc.expected)
		}
	}
}

func TestParseGraphicsProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected GraphicsProtocol
		wantErr  bool
	}{
		{"auto", ProtocolAuto, false},
		{"", ProtocolAuto, false},
		{"iterm2", ProtocolITerm2, false},
		{"iterm", ProtocolITerm2, false},
		{"osc1337", ProtocolITerm2, false},
		{"kitty", ProtocolKitty, false},
		{"sixel", ProtocolSixel, false},
		{"none", ProtocolNone, false},
		{"text", ProtocolNone, false},
		{"ascii", ProtocolNone, false},
		{"invalid", ProtocolNone, true},
	}

	for _, tc := range tests {
		got, err := ParseGraphicsProtocol(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseGraphicsProtocol(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
		}
		if got != tc.expected {
			t.Errorf("ParseGraphicsProtocol(%q) = %v, expected %v", tc.input, got, tc.expected)
		}
	}
}

func TestDetectGraphicsProtocol(t *testing.T) {
	tests := []struct {
		name            string
		env             mapEnv
		allowCompatible bool
		expected        GraphicsProtocol
	}{
		{
			name:     "Native Kitty via KITTY_WINDOW_ID",
			env:      mapEnv{"KITTY_WINDOW_ID": "1"},
			expected: ProtocolKitty,
		},
		{
			name:     "Native Kitty via TERM",
			env:      mapEnv{"TERM": "xterm-kitty"},
			expected: ProtocolKitty,
		},
		{
			name:     "Ghostty terminal",
			env:      mapEnv{"TERM_PROGRAM": "ghostty"},
			expected: ProtocolKitty,
		},
		{
			name:     "iTerm2 via TERM_PROGRAM",
			env:      mapEnv{"TERM_PROGRAM": "iTerm.app"},
			expected: ProtocolITerm2,
		},
		{
			name:     "iTerm2 via LC_TERMINAL",
			env:      mapEnv{"LC_TERMINAL": "iTerm2"},
			expected: ProtocolITerm2,
		},
		{
			name:            "WezTerm compatible enabled",
			env:             mapEnv{"TERM_PROGRAM": "wezterm"},
			allowCompatible: true,
			expected:        ProtocolITerm2,
		},
		{
			name:            "WezTerm compatible disabled",
			env:             mapEnv{"TERM_PROGRAM": "wezterm"},
			allowCompatible: false,
			expected:        ProtocolNone,
		},
		{
			name:     "Foot terminal Sixel",
			env:      mapEnv{"TERM": "foot"},
			expected: ProtocolSixel,
		},
		{
			name:     "mlterm Sixel",
			env:      mapEnv{"TERM": "mlterm"},
			expected: ProtocolSixel,
		},
		{
			name:     "SIXEL_SUPPORT variable set",
			env:      mapEnv{"SIXEL_SUPPORT": "1"},
			expected: ProtocolSixel,
		},
		{
			name:     "Standard Terminal without graphics",
			env:      mapEnv{"TERM": "xterm-256color"},
			expected: ProtocolNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectGraphicsProtocol(tc.env, tc.allowCompatible)
			if got != tc.expected {
				t.Errorf("DetectGraphicsProtocol() = %v, expected %v", got, tc.expected)
			}
		})
	}
}

func TestSupportsGraphicsProtocol(t *testing.T) {
	itermEnv := mapEnv{"TERM_PROGRAM": "iTerm.app"}
	kittyEnv := mapEnv{"KITTY_WINDOW_ID": "42"}
	footEnv := mapEnv{"TERM": "foot"}
	plainEnv := mapEnv{"TERM": "xterm"}

	if !SupportsGraphicsProtocol(itermEnv, ProtocolITerm2, true) {
		t.Error("expected iTerm2 to be supported in itermEnv")
	}
	if !SupportsGraphicsProtocol(kittyEnv, ProtocolKitty, true) {
		t.Error("expected Kitty to be supported in kittyEnv")
	}
	if !SupportsGraphicsProtocol(footEnv, ProtocolSixel, true) {
		t.Error("expected Sixel to be supported in footEnv")
	}
	if SupportsGraphicsProtocol(plainEnv, ProtocolKitty, true) {
		t.Error("expected Kitty to be unsupported in plainEnv")
	}
	if !SupportsGraphicsProtocol(plainEnv, ProtocolNone, true) {
		t.Error("expected ProtocolNone to always be supported")
	}
	if !SupportsGraphicsProtocol(footEnv, ProtocolAuto, true) {
		t.Error("expected ProtocolAuto to be supported in footEnv")
	}
	if SupportsGraphicsProtocol(plainEnv, ProtocolAuto, true) {
		t.Error("expected ProtocolAuto to be false in plainEnv")
	}
}

func TestFormatKittyImage_SmallPayload(t *testing.T) {
	smallData := []byte("hello-png-image")
	output := FormatKittyImage(smallData, 60, 30, false)

	if !strings.HasPrefix(output, "\033_G") {
		t.Errorf("expected Kitty APC prefix \\033_G, got %q", output)
	}
	if !strings.Contains(output, "a=T") || !strings.Contains(output, "f=100") {
		t.Errorf("expected a=T and f=100 controls in output: %q", output)
	}
	if !strings.Contains(output, "m=0") {
		t.Errorf("expected m=0 for single chunk, got %q", output)
	}
	if !strings.Contains(output, "c=60") || !strings.Contains(output, "r=30") {
		t.Errorf("expected cell dimension controls c=60,r=30, got %q", output)
	}
	if !strings.Contains(output, "q=2") {
		t.Errorf("expected quiet mode q=2, got %q", output)
	}
	if !strings.HasSuffix(output, "\033\\\n") {
		t.Errorf("expected string terminator \\033\\ followed by newline, got %q", output)
	}
}

func TestFormatKittyImage_LargePayload_Chunking(t *testing.T) {
	// Create payload > 4096 bytes base64 (e.g. 5000 raw bytes -> ~6668 base64 bytes)
	largeData := bytes.Repeat([]byte("A"), 5000)
	output := FormatKittyImage(largeData, 0, 0, false)

	// Should contain first chunk with m=1 and subsequent chunk with m=0
	if !strings.Contains(output, "m=1") {
		t.Errorf("expected m=1 in chunked Kitty output")
	}
	if !strings.Contains(output, "m=0") {
		t.Errorf("expected m=0 in final chunk of Kitty output")
	}

	chunks := strings.Split(output, "\033_G")
	// Index 0 is empty before the first escape code, so we expect at least 2 chunks
	if len(chunks) < 3 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks)-1)
	}
}

func TestFormatKittyImage_TmuxPassthrough(t *testing.T) {
	smallData := []byte("tmux-test-png")
	output := FormatKittyImage(smallData, 0, 0, true)

	if !strings.HasPrefix(output, "\033Ptmux;\033\033_G") {
		t.Errorf("expected tmux passthrough prefix, got %q", output)
	}
	if !strings.Contains(output, "\033\033\\\033\\") {
		t.Errorf("expected doubled escape terminator in tmux passthrough, got %q", output)
	}
}

func createTestImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
			}
		}
	}
	return img
}

func TestEncodeSixel(t *testing.T) {
	img := createTestImage(12, 12)
	sixel, err := EncodeSixel(img, false)
	if err != nil {
		t.Fatalf("EncodeSixel failed: %v", err)
	}

	// Sixel DCS sequence header
	if !strings.HasPrefix(sixel, "\033Pq\"1;1;12;12") {
		t.Errorf("expected Sixel DCS header, got: %s", sixel[:min(len(sixel), 30)])
	}
	// Palette definitions
	if !strings.Contains(sixel, "#0;2;") {
		t.Errorf("expected color palette definition in Sixel output")
	}
	// Sixel terminator
	if !strings.HasSuffix(sixel, "\033\\") {
		t.Errorf("expected Sixel string terminator \\033\\")
	}

	// Test tmux passthrough
	tmuxSixel, err := EncodeSixel(img, true)
	if err != nil {
		t.Fatalf("EncodeSixel with tmux failed: %v", err)
	}
	if !strings.HasPrefix(tmuxSixel, "\033Ptmux;\033\033Pq") {
		t.Errorf("expected tmux wrapped Sixel header, got: %s", tmuxSixel[:min(len(tmuxSixel), 30)])
	}
}

func TestFormatSixelImage_ValidPNG(t *testing.T) {
	img := createTestImage(8, 8)
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	sixel, err := FormatSixelImage(buf.Bytes(), false)
	if err != nil {
		t.Fatalf("FormatSixelImage failed: %v", err)
	}

	if !strings.HasPrefix(sixel, "\033Pq\"1;1;8;8") {
		t.Errorf("unexpected sixel output: %s", sixel[:min(len(sixel), 30)])
	}
}

func TestFormatSixelImage_InvalidData(t *testing.T) {
	_, err := FormatSixelImage([]byte("not-an-image"), false)
	if err == nil {
		t.Fatal("expected error decoding invalid image data for sixel, got nil")
	}
}

func TestFormatTerminalImage(t *testing.T) {
	img := createTestImage(4, 4)
	buf := new(bytes.Buffer)
	_ = png.Encode(buf, img)
	pngBytes := buf.Bytes()

	opts := ImageFormatOptions{
		Width:               "60cell",
		Height:              "30cell",
		PreserveAspectRatio: true,
	}

	// Kitty
	kittyOut, err := FormatTerminalImage(ProtocolKitty, pngBytes, opts)
	if err != nil {
		t.Fatalf("FormatTerminalImage(ProtocolKitty) err: %v", err)
	}
	if !strings.Contains(kittyOut, "\033_G") || !strings.Contains(kittyOut, "c=60") {
		t.Errorf("unexpected Kitty output: %q", kittyOut)
	}

	// iTerm2
	itermOut, err := FormatTerminalImage(ProtocolITerm2, pngBytes, opts)
	if err != nil {
		t.Fatalf("FormatTerminalImage(ProtocolITerm2) err: %v", err)
	}
	if !strings.Contains(itermOut, "\033]1337;File=") {
		t.Errorf("unexpected iTerm2 output: %q", itermOut)
	}

	// Sixel
	sixelOut, err := FormatTerminalImage(ProtocolSixel, pngBytes, opts)
	if err != nil {
		t.Fatalf("FormatTerminalImage(ProtocolSixel) err: %v", err)
	}
	if !strings.Contains(sixelOut, "\033Pq") {
		t.Errorf("unexpected Sixel output: %q", sixelOut)
	}

	// None / Unsupported
	_, err = FormatTerminalImage(ProtocolNone, pngBytes, opts)
	if err == nil {
		t.Fatal("expected error for ProtocolNone, got nil")
	}
}

func TestPrinter_MultiProtocol(t *testing.T) {
	img := createTestImage(10, 10)
	buf := new(bytes.Buffer)
	_ = png.Encode(buf, img)
	mockImg := &mockImageRenderer{data: buf.Bytes()}

	// 1. Explicit ProtocolKitty
	pKitty := New(
		WithMode(ModeImage),
		WithGraphicsProtocol(ProtocolKitty),
		WithImageRenderer(mockImg),
	)
	resKitty, err := pKitty.Render(context.Background(), "graph TD\n A-->B")
	if err != nil {
		t.Fatalf("pKitty.Render err: %v", err)
	}
	if resKitty.Protocol != ProtocolKitty {
		t.Errorf("expected ProtocolKitty, got %v", resKitty.Protocol)
	}
	if !strings.HasPrefix(resKitty.Output, "\033_G") {
		t.Errorf("expected Kitty APC sequence, got: %q", resKitty.Output)
	}

	// 2. Explicit ProtocolSixel
	pSixel := New(
		WithMode(ModeImage),
		WithGraphicsProtocol(ProtocolSixel),
		WithImageRenderer(mockImg),
	)
	resSixel, err := pSixel.Render(context.Background(), "graph TD\n A-->B")
	if err != nil {
		t.Fatalf("pSixel.Render err: %v", err)
	}
	if resSixel.Protocol != ProtocolSixel {
		t.Errorf("expected ProtocolSixel, got %v", resSixel.Protocol)
	}
	if !strings.HasPrefix(resSixel.Output, "\033Pq") {
		t.Errorf("expected Sixel DCS sequence, got: %q", resSixel.Output)
	}

	// 3. Auto detection for Kitty environment
	pAutoKitty := New(
		WithMode(ModeAuto),
		WithForceTTY(true),
		WithTerminalEnv(mapEnv{"KITTY_WINDOW_ID": "1"}),
		WithImageRenderer(mockImg),
	)
	resAutoKitty, err := pAutoKitty.Render(context.Background(), "graph TD\n A-->B")
	if err != nil {
		t.Fatalf("pAutoKitty.Render err: %v", err)
	}
	if resAutoKitty.Protocol != ProtocolKitty {
		t.Errorf("expected detected ProtocolKitty, got %v", resAutoKitty.Protocol)
	}

	// 4. Auto detection for Sixel environment (Foot)
	pAutoFoot := New(
		WithMode(ModeAuto),
		WithForceTTY(true),
		WithTerminalEnv(mapEnv{"TERM": "foot"}),
		WithImageRenderer(mockImg),
	)
	resAutoFoot, err := pAutoFoot.Render(context.Background(), "graph TD\n A-->B")
	if err != nil {
		t.Fatalf("pAutoFoot.Render err: %v", err)
	}
	if resAutoFoot.Protocol != ProtocolSixel {
		t.Errorf("expected detected ProtocolSixel, got %v", resAutoFoot.Protocol)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

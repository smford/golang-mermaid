package mermaid

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestParseDA1Response(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedSixel bool
		expectedKitty bool
	}{
		{
			name:          "DEC Sixel supported (parameter 4 present)",
			input:         "\033[?62;1;2;4;6;7;8;9c",
			expectedSixel: true,
			expectedKitty: false,
		},
		{
			name:          "Standard VT220 without Sixel",
			input:         "\033[?62;1;2;6;7;8;9c",
			expectedSixel: false,
			expectedKitty: false,
		},
		{
			name:          "Kitty Graphics APC response",
			input:         "\033_Gi=1;OK\033\\",
			expectedSixel: false,
			expectedKitty: true,
		},
		{
			name:          "Empty response",
			input:         "",
			expectedSixel: false,
			expectedKitty: false,
		},
		{
			name:          "Malformed response",
			input:         "random text without escape sequences",
			expectedSixel: false,
			expectedKitty: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caps := ParseDA1Response(tc.input)
			if caps.SupportsSixel != tc.expectedSixel {
				t.Errorf("expected SupportsSixel=%v, got %v", tc.expectedSixel, caps.SupportsSixel)
			}
			if caps.SupportsKitty != tc.expectedKitty {
				t.Errorf("expected SupportsKitty=%v, got %v", tc.expectedKitty, caps.SupportsKitty)
			}
		})
	}
}

func TestProbeTerminalStreams_Success(t *testing.T) {
	out := new(bytes.Buffer)
	inReader, inWriter := io.Pipe()

	go func() {
		// Mock terminal responding to DA1 query
		_, _ = inWriter.Write([]byte("\033[?64;1;2;4;9;15;18;21;22c"))
		_ = inWriter.Close()
	}()

	caps := ProbeTerminalStreams(inReader, out, 100*time.Millisecond)

	// Verify query was sent
	if out.String() != "\033[c" {
		t.Errorf("expected '\\033[c' query sent to terminal, got: %q", out.String())
	}
	// Verify Sixel detected
	if !caps.SupportsSixel {
		t.Errorf("expected SupportsSixel=true from mock response")
	}
}

func TestProbeTerminalStreams_Timeout(t *testing.T) {
	out := new(bytes.Buffer)
	inReader, inWriter := io.Pipe()
	defer inWriter.Close()

	// Does not write anything to inWriter
	caps := ProbeTerminalStreams(inReader, out, 20*time.Millisecond)

	if caps.SupportsSixel || caps.SupportsKitty {
		t.Errorf("expected empty capabilities on timeout, got: %+v", caps)
	}
}

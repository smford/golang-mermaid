package mermaid

import (
	"encoding/base64"
	"strings"
	"testing"
)

type mapEnv map[string]string

func (m mapEnv) Getenv(key string) string {
	return m[key]
}

func TestIsITerm2Env(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "iTerm via TERM_PROGRAM",
			env:  map[string]string{"TERM_PROGRAM": "iTerm.app"},
			want: true,
		},
		{
			name: "iTerm via LC_TERMINAL",
			env:  map[string]string{"LC_TERMINAL": "iTerm2"},
			want: true,
		},
		{
			name: "iTerm via ITERM_SESSION_ID",
			env:  map[string]string{"ITERM_SESSION_ID": "w0t0p0:some-session"},
			want: true,
		},
		{
			name: "Apple Terminal",
			env:  map[string]string{"TERM_PROGRAM": "Apple_Terminal"},
			want: false,
		},
		{
			name: "Standard xterm",
			env:  map[string]string{"TERM": "xterm-256color"},
			want: false,
		},
		{
			name: "Empty environment",
			env:  map[string]string{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsITerm2Env(mapEnv(tt.env))
			if got != tt.want {
				t.Errorf("IsITerm2Env() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSupportsITerm2ImagesEnv(t *testing.T) {
	tests := []struct {
		name            string
		env             map[string]string
		allowCompatible bool
		want            bool
	}{
		{
			name:            "iTerm.app",
			env:             map[string]string{"TERM_PROGRAM": "iTerm.app"},
			allowCompatible: false,
			want:            true,
		},
		{
			name:            "WezTerm with compatible allowed",
			env:             map[string]string{"TERM_PROGRAM": "WezTerm"},
			allowCompatible: true,
			want:            true,
		},
		{
			name:            "WezTerm without compatible allowed",
			env:             map[string]string{"TERM_PROGRAM": "WezTerm"},
			allowCompatible: false,
			want:            false,
		},
		{
			name:            "Ghostty with compatible allowed",
			env:             map[string]string{"TERM_PROGRAM": "ghostty"},
			allowCompatible: true,
			want:            true,
		},
		{
			name:            "Apple_Terminal",
			env:             map[string]string{"TERM_PROGRAM": "Apple_Terminal"},
			allowCompatible: true,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportsITerm2ImagesEnv(mapEnv(tt.env), tt.allowCompatible)
			if got != tt.want {
				t.Errorf("SupportsITerm2ImagesEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTmuxEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "TMUX env variable set",
			env:  map[string]string{"TMUX": "/tmp/tmux-501/default,1234,0"},
			want: true,
		},
		{
			name: "TERM set to tmux-256color",
			env:  map[string]string{"TERM": "tmux-256color"},
			want: true,
		},
		{
			name: "TERM set to screen-256color",
			env:  map[string]string{"TERM": "screen-256color"},
			want: true,
		},
		{
			name: "Normal terminal",
			env:  map[string]string{"TERM": "xterm-256color"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTmuxEnv(mapEnv(tt.env))
			if got != tt.want {
				t.Errorf("IsTmuxEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatITerm2Image(t *testing.T) {
	fakeData := []byte("FAKE_PNG_BINARY_DATA")
	expectedPayload := base64.StdEncoding.EncodeToString(fakeData)

	opts := ImageFormatOptions{
		Inline:              true,
		Width:               "800px",
		Height:              "400px",
		PreserveAspectRatio: true,
		FileName:            "diagram.png",
		TmuxPassthrough:     false,
	}

	seq := FormatITerm2Image(fakeData, opts)

	if !strings.HasPrefix(seq, "\033]1337;File=") {
		t.Fatalf("expected OSC prefix, got: %q", seq)
	}
	if !strings.HasSuffix(seq, "\a") {
		t.Fatalf("expected BEL terminator, got: %q", seq)
	}
	if !strings.Contains(seq, "inline=1") {
		t.Errorf("expected inline=1 in %q", seq)
	}
	if !strings.Contains(seq, "width=800px") {
		t.Errorf("expected width=800px in %q", seq)
	}
	if !strings.Contains(seq, "height=400px") {
		t.Errorf("expected height=400px in %q", seq)
	}
	if !strings.Contains(seq, "preserveAspectRatio=1") {
		t.Errorf("expected preserveAspectRatio=1 in %q", seq)
	}
	if !strings.Contains(seq, expectedPayload) {
		t.Errorf("expected payload %q in %q", expectedPayload, seq)
	}

	// Test tmux wrapping
	opts.TmuxPassthrough = true
	tmuxSeq := FormatITerm2Image(fakeData, opts)
	if !strings.HasPrefix(tmuxSeq, "\033Ptmux;") {
		t.Errorf("expected tmux DCS prefix, got: %q", tmuxSeq)
	}
	if !strings.HasSuffix(tmuxSeq, "\033\\") {
		t.Errorf("expected tmux string terminator, got: %q", tmuxSeq)
	}
}

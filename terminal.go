package mermaid

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// TerminalEnv abstracts environment variable lookup for testing and custom environments.
type TerminalEnv interface {
	Getenv(key string) string
}

// osEnv implements TerminalEnv using standard os.Getenv.
type osEnv struct{}

func (osEnv) Getenv(key string) string {
	return os.Getenv(key)
}

// DefaultTerminalEnv is the default environment lookup implementation.
var DefaultTerminalEnv TerminalEnv = osEnv{}

// IsITerm2 reports whether the current process is running directly within iTerm2.
// It inspects TERM_PROGRAM, ITERM_SESSION_ID, and LC_TERMINAL.
func IsITerm2() bool {
	return IsITerm2Env(DefaultTerminalEnv)
}

// IsITerm2Env reports whether the given environment represents an iTerm2 terminal.
func IsITerm2Env(env TerminalEnv) bool {
	if env == nil {
		env = DefaultTerminalEnv
	}
	if env.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return true
	}
	if env.Getenv("LC_TERMINAL") == "iTerm2" {
		return true
	}
	if env.Getenv("ITERM_SESSION_ID") != "" {
		return true
	}
	return false
}

// SupportsITerm2Images reports whether the current terminal supports the iTerm2
// inline image protocol (OSC 1337). If allowCompatible is true, terminals known
// to implement the iTerm2 image protocol (such as WezTerm and Ghostty) are also accepted.
func SupportsITerm2Images(allowCompatible bool) bool {
	return SupportsITerm2ImagesEnv(DefaultTerminalEnv, allowCompatible)
}

// SupportsITerm2ImagesEnv checks image support using the supplied TerminalEnv.
func SupportsITerm2ImagesEnv(env TerminalEnv, allowCompatible bool) bool {
	if env == nil {
		env = DefaultTerminalEnv
	}

	if IsITerm2Env(env) {
		return true
	}

	if allowCompatible {
		prog := strings.ToLower(env.Getenv("TERM_PROGRAM"))
		switch prog {
		case "wezterm", "ghostty", "mintty":
			return true
		}
	}

	return false
}

// IsTerminal reports whether the provided writer is connected to an interactive terminal (TTY).
// If w is nil or not an *os.File, it returns false.
func IsTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// IsTmux reports whether the current environment is running inside tmux.
func IsTmux() bool {
	return IsTmuxEnv(DefaultTerminalEnv)
}

// IsTmuxEnv checks if the given environment indicates a tmux session.
func IsTmuxEnv(env TerminalEnv) bool {
	if env == nil {
		env = DefaultTerminalEnv
	}
	if env.Getenv("TMUX") != "" {
		return true
	}
	termVal := env.Getenv("TERM")
	return strings.HasPrefix(termVal, "tmux") || strings.HasPrefix(termVal, "screen")
}

// ImageFormatOptions controls how the iTerm2 OSC 1337 escape sequence is generated.
type ImageFormatOptions struct {
	// Inline specifies whether the image should be displayed inline (1) or downloaded (0).
	// Default is true.
	Inline bool

	// Width specifies the image display width (e.g. "auto", "100%", "800px", "60cell").
	Width string

	// Height specifies the image display height (e.g. "auto", "100%", "400px", "30cell").
	Height string

	// PreserveAspectRatio determines whether the image aspect ratio is preserved.
	// Default is true.
	PreserveAspectRatio bool

	// FileName is an optional filename associated with the image metadata.
	FileName string

	// TmuxPassthrough wraps the OSC sequence in a tmux DCS passthrough escape sequence.
	TmuxPassthrough bool
}

// DefaultImageFormatOptions returns the recommended default image formatting options.
func DefaultImageFormatOptions() ImageFormatOptions {
	return ImageFormatOptions{
		Inline:              true,
		Width:               "auto",
		Height:              "auto",
		PreserveAspectRatio: true,
		TmuxPassthrough:     IsTmux(),
	}
}

// FormatITerm2Image encodes raw image data (e.g. PNG, JPEG) into the iTerm2
// OSC 1337 inline image escape sequence.
func FormatITerm2Image(imageData []byte, opts ImageFormatOptions) string {
	var args []string

	// inline=1
	inlineVal := "0"
	if opts.Inline {
		inlineVal = "1"
	}
	args = append(args, fmt.Sprintf("inline=%s", inlineVal))

	// size=<bytes>
	if len(imageData) > 0 {
		args = append(args, fmt.Sprintf("size=%d", len(imageData)))
	}

	// name=<base64-filename>
	if opts.FileName != "" {
		encodedName := base64.StdEncoding.EncodeToString([]byte(opts.FileName))
		args = append(args, fmt.Sprintf("name=%s", encodedName))
	}

	// width
	if opts.Width != "" {
		args = append(args, fmt.Sprintf("width=%s", opts.Width))
	}

	// height
	if opts.Height != "" {
		args = append(args, fmt.Sprintf("height=%s", opts.Height))
	}

	// preserveAspectRatio
	if opts.PreserveAspectRatio {
		args = append(args, "preserveAspectRatio=1")
	} else {
		args = append(args, "preserveAspectRatio=0")
	}

	encodedPayload := base64.StdEncoding.EncodeToString(imageData)
	argString := strings.Join(args, ";")

	// Standard iTerm2 OSC 1337 format: \033]1337;File=<args>:<payload>\a
	osc := fmt.Sprintf("\033]1337;File=%s:%s\a", argString, encodedPayload)

	if opts.TmuxPassthrough {
		// Inside tmux, wrap in DCS passthrough: \033Ptmux;\033<sequence-with-doubled-esc>\033\\
		escapedOSC := strings.ReplaceAll(osc, "\033", "\033\033")
		return fmt.Sprintf("\033Ptmux;%s\033\\", escapedOSC)
	}

	return osc
}

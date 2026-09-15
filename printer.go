package mermaid

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RenderResult encapsulates the result of a diagram rendering operation.
type RenderResult struct {
	// Mode is the actual render mode used (e.g. ModeImage, ModeUnicode, ModeASCII).
	Mode RenderMode

	// ImageData contains raw image bytes (PNG) if image rendering was performed.
	ImageData []byte

	// Output is the formatted terminal payload (either OSC 1337 escape sequence or text art).
	Output string

	// FallbackOccurred reports whether a fallback from image to text was triggered.
	FallbackOccurred bool

	// FallbackReason describes why the fallback occurred, if applicable.
	FallbackReason string

	// Duration is the total time spent rendering.
	Duration time.Duration
}

// Printer coordinates rendering and terminal output of Mermaid diagrams.
type Printer struct {
	config Config
}

// New creates a new Printer configured with the provided options.
func New(opts ...Option) *Printer {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Initialize default renderers if not supplied
	if cfg.ImageRenderer == nil {
		cfg.ImageRenderer = NewResilientImageRendererWithScale("", cfg.Timeout, cfg.Scale)
	}
	if cfg.TextRenderer == nil {
		cfg.TextRenderer = NewFallbackTextRendererFromConfig(cfg)
	}

	return &Printer{config: cfg}
}

// Print renders the Mermaid diagram source string to the configured writer.
func (p *Printer) Print(mermaidSource string) error {
	return p.PrintContext(context.Background(), mermaidSource)
}

// PrintContext renders the Mermaid diagram source string with a context for cancellation.
func (p *Printer) PrintContext(ctx context.Context, mermaidSource string) error {
	result, err := p.Render(ctx, mermaidSource)
	if err != nil {
		return err
	}

	_, err = io.WriteString(p.config.Writer, result.Output)
	if err != nil {
		return fmt.Errorf("write diagram output: %w", err)
	}

	// Ensure newline after diagram output
	if !strings.HasSuffix(result.Output, "\n") {
		if _, err := io.WriteString(p.config.Writer, "\n"); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	return nil
}

// PrintFile reads a Mermaid file from disk and prints it to the configured writer.
func (p *Printer) PrintFile(filePath string) error {
	return p.PrintFileContext(context.Background(), filePath)
}

// PrintFileContext reads a Mermaid file and prints it with context support.
func (p *Printer) PrintFileContext(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read mermaid file %q: %w", filePath, err)
	}

	// If no filename was set in config, use basename of the file
	pCopy := *p
	if pCopy.config.Width == "" {
		pCopy.config.Width = "auto"
	}

	return pCopy.PrintContext(ctx, string(data))
}

// Render processes the Mermaid diagram and produces a RenderResult without writing to output.
func (p *Printer) Render(ctx context.Context, mermaidSource string) (*RenderResult, error) {
	start := time.Now()
	mermaidSource = strings.TrimSpace(mermaidSource)
	if mermaidSource == "" {
		return nil, ErrEmptySource
	}

	// Apply timeout to context if not already bounded
	var cancel context.CancelFunc
	if p.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, p.config.Timeout)
		defer cancel()
	}

	targetMode := p.config.Mode
	var fallbackReason string
	fallbackOccurred := false

	// Determine if image rendering should be attempted
	shouldAttemptImage := false
	switch targetMode {
	case ModeImage:
		shouldAttemptImage = true
	case ModeAuto:
		// Check terminal capability
		supported := SupportsITerm2ImagesEnv(p.config.TerminalEnv, p.config.AllowCompatibleTerminals)
		isTTY := p.config.ForceTTY || IsTerminal(p.config.Writer)

		if !supported {
			fallbackReason = "terminal does not support iTerm2 inline images"
			fallbackOccurred = true
			targetMode = ModeUnicode
		} else if !isTTY {
			fallbackReason = "output destination is not an interactive terminal (TTY)"
			fallbackOccurred = true
			targetMode = ModeUnicode
		} else {
			shouldAttemptImage = true
		}
	case ModeASCII, ModeUnicode:
		shouldAttemptImage = false
	}

	// Attempt image rendering if eligible
	var imgData []byte
	if shouldAttemptImage {
		var err error
		imgData, err = p.config.ImageRenderer.RenderImage(ctx, mermaidSource)
		if err != nil {
			if p.config.DisableFallback {
				return nil, fmt.Errorf("image rendering failed: %w", err)
			}
			fallbackReason = fmt.Sprintf("image rendering failed: %v", err)
			fallbackOccurred = true
			targetMode = ModeUnicode
		}
	}

	// Notify SRE observability hook if fallback occurred
	if fallbackOccurred && p.config.OnFallback != nil {
		p.config.OnFallback(fallbackReason, nil)
	}

	// Render Image output
	if targetMode == ModeImage || (shouldAttemptImage && !fallbackOccurred) {
		fmtOpts := ImageFormatOptions{
			Inline:              true,
			Width:               p.config.Width,
			Height:              p.config.Height,
			PreserveAspectRatio: p.config.PreserveAspectRatio,
			TmuxPassthrough:     IsTmuxEnv(p.config.TerminalEnv),
		}

		output := FormatITerm2Image(imgData, fmtOpts)
		return &RenderResult{
			Mode:             ModeImage,
			ImageData:        imgData,
			Output:           output,
			FallbackOccurred: false,
			Duration:         time.Since(start),
		}, nil
	}

	// Render Text / ASCII output
	text, err := p.config.TextRenderer.RenderText(ctx, mermaidSource)
	if err != nil {
		return nil, fmt.Errorf("text rendering failed: %w", err)
	}

	actualMode := ModeUnicode
	if targetMode == ModeASCII {
		actualMode = ModeASCII
	}

	return &RenderResult{
		Mode:             actualMode,
		Output:           text,
		FallbackOccurred: fallbackOccurred,
		FallbackReason:   fallbackReason,
		Duration:         time.Since(start),
	}, nil
}

// Print renders the Mermaid diagram source using default settings and writes to os.Stdout.
func Print(mermaidSource string, opts ...Option) error {
	return New(opts...).Print(mermaidSource)
}

// PrintContext renders the Mermaid diagram source with context using default settings.
func PrintContext(ctx context.Context, mermaidSource string, opts ...Option) error {
	return New(opts...).PrintContext(ctx, mermaidSource)
}

// PrintFile reads and renders a Mermaid file using default settings and writes to os.Stdout.
func PrintFile(filePath string, opts ...Option) error {
	return New(opts...).PrintFile(filePath)
}

// PrintFileContext reads and renders a Mermaid file with context using default settings.
func PrintFileContext(ctx context.Context, filePath string, opts ...Option) error {
	return New(opts...).PrintFileContext(ctx, filePath)
}

// Render parses and formats the Mermaid diagram without writing to an output stream.
func Render(ctx context.Context, mermaidSource string, opts ...Option) (*RenderResult, error) {
	return New(opts...).Render(ctx, mermaidSource)
}

// RenderFile reads a Mermaid file and renders it without writing to an output stream.
func RenderFile(ctx context.Context, filePath string, opts ...Option) (*RenderResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read mermaid file %q: %w", filePath, err)
	}
	p := New(opts...)
	if p.config.Width == "" {
		p.config.Width = "auto"
	}
	_ = filepath.Base(filePath)
	return p.Render(ctx, string(data))
}

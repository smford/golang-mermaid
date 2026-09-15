package mermaid

import (
	"io"
	"os"
	"time"
)

// RenderMode specifies how diagrams should be rendered.
type RenderMode int

const (
	// ModeAuto automatically determines whether to use iTerm2 inline images
	// or ASCII/Unicode fallback based on terminal capabilities and TTY state.
	ModeAuto RenderMode = iota

	// ModeImage forces iTerm2 inline image rendering.
	// If image generation fails, it falls back to ASCII unless DisableFallback is true.
	ModeImage

	// ModeASCII forces text-based diagram rendering using pure 7-bit ASCII characters.
	ModeASCII

	// ModeUnicode forces text-based diagram rendering using Unicode box-drawing characters.
	ModeUnicode
)

func (m RenderMode) String() string {
	switch m {
	case ModeAuto:
		return "auto"
	case ModeImage:
		return "image"
	case ModeASCII:
		return "ascii"
	case ModeUnicode:
		return "unicode"
	default:
		return "unknown"
	}
}

// FallbackFunc is a callback invoked when an image render falls back to text/ASCII.
type FallbackFunc func(reason string, err error)

// Config contains all configuration options for diagram printing.
type Config struct {
	// Mode specifies the rendering mode. Default is ModeAuto.
	Mode RenderMode

	// Writer specifies the destination stream. Default is os.Stdout.
	Writer io.Writer

	// Width specifies the image width in iTerm2 (e.g. "auto", "100%", "800px", "60cell").
	Width string

	// Height specifies the image height in iTerm2 (e.g. "auto", "100%", "400px", "30cell").
	Height string

	// PreserveAspectRatio determines whether aspect ratio is preserved in iTerm2.
	PreserveAspectRatio bool

	// Scale specifies the rasterization scale factor for image rendering (e.g. 1.0, 2.0 for Retina/HiDPI).
	// Default is 1.0.
	Scale float64

	// AllowCompatibleTerminals allows terminals that support iTerm2 OSC 1337
	// (such as WezTerm, Ghostty, and mintty) to render images in ModeAuto.
	// Default is true.
	AllowCompatibleTerminals bool

	// DisableFallback prevents falling back to ASCII/Unicode when image rendering fails.
	DisableFallback bool

	// ForceTTY overrides the TTY detection check (useful for automated testing or virtual TTYs).
	ForceTTY bool

	// Timeout specifies the maximum time allowed for rendering operations (e.g. remote HTTP calls).
	// Default is 10 seconds.
	Timeout time.Duration

	// Theme specifies diagram color/theme if supported by the renderer (e.g. "default", "dark", "slate", "blueprint", "neon").
	Theme string

	// Columns specifies the character column width for text diagram layout.
	// When <= 0, automatically detects terminal width or uses a generous 120 default.
	Columns int

	// PaddingX specifies horizontal padding inside node boxes.
	PaddingX int

	// PaddingY specifies vertical padding inside node boxes.
	PaddingY int

	// SharpEdges uses sharp corners (┌──┐) instead of rounded corners (╭──╮).
	SharpEdges bool

	// Hyperlinks enables OSC 8 clickable terminal hyperlinks for diagrams with click events.
	Hyperlinks bool

	// BoxFrame wraps the text diagram in an outer decorative card border.
	BoxFrame bool

	// Title is an optional title displayed in the diagram frame or header.
	Title string

	// ImageRenderer is the backend used to generate image bytes from Mermaid syntax.
	ImageRenderer ImageRenderer

	// TextRenderer is the backend used to generate ASCII/Unicode text from Mermaid syntax.
	TextRenderer TextRenderer

	// OnFallback is an optional hook called whenever fallback occurs, providing SRE observability.
	OnFallback FallbackFunc

	// TerminalEnv provides environment variable lookup. Defaults to osEnv.
	TerminalEnv TerminalEnv
}

// Option configures a Config struct.
type Option func(*Config)

// DefaultConfig returns the default printer configuration.
func DefaultConfig() Config {
	return Config{
		Mode:                     ModeAuto,
		Writer:                   os.Stdout,
		Width:                    "auto",
		Height:                   "auto",
		PreserveAspectRatio:      true,
		Scale:                    1.0,
		AllowCompatibleTerminals: true,
		DisableFallback:          false,
		ForceTTY:                 false,
		Timeout:                  10 * time.Second,
		Theme:                    "default",
		Columns:                  0, // Auto-detect
		PaddingX:                 1,
		PaddingY:                 0,
		SharpEdges:               false,
		Hyperlinks:               false,
		BoxFrame:                 false,
		Title:                    "",
		TerminalEnv:              DefaultTerminalEnv,
	}
}

// WithMode sets the rendering mode (ModeAuto, ModeImage, ModeASCII, ModeUnicode).
func WithMode(mode RenderMode) Option {
	return func(c *Config) {
		c.Mode = mode
	}
}

// WithWriter sets the destination io.Writer.
func WithWriter(w io.Writer) Option {
	return func(c *Config) {
		if w != nil {
			c.Writer = w
		}
	}
}

// WithWidth sets the display width in iTerm2 (e.g. "auto", "100%", "800px", "60cell").
func WithWidth(width string) Option {
	return func(c *Config) {
		c.Width = width
	}
}

// WithHeight sets the display height in iTerm2 (e.g. "auto", "100%", "400px", "30cell").
func WithHeight(height string) Option {
	return func(c *Config) {
		c.Height = height
	}
}

// WithPreserveAspectRatio specifies whether to preserve aspect ratio in iTerm2.
func WithPreserveAspectRatio(preserve bool) Option {
	return func(c *Config) {
		c.PreserveAspectRatio = preserve
	}
}

// WithScale sets the image rasterization scale factor (e.g. 1.0, 2.0 for Retina/HiDPI displays).
func WithScale(scale float64) Option {
	return func(c *Config) {
		if scale > 0 {
			c.Scale = scale
		}
	}
}

// WithAllowCompatibleTerminals controls whether non-iTerm2 terminals that support
// OSC 1337 (like WezTerm, Ghostty) should be treated as image-capable.
func WithAllowCompatibleTerminals(allow bool) Option {
	return func(c *Config) {
		c.AllowCompatibleTerminals = allow
	}
}

// WithDisableFallback disables automatic fallback to ASCII when image rendering fails.
func WithDisableFallback(disable bool) Option {
	return func(c *Config) {
		c.DisableFallback = disable
	}
}

// WithForceTTY forces the printer to treat the destination as an interactive TTY.
func WithForceTTY(force bool) Option {
	return func(c *Config) {
		c.ForceTTY = force
	}
}

// WithTimeout sets the timeout duration for network requests or external CLI rendering.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		if timeout > 0 {
			c.Timeout = timeout
		}
	}
}

// WithTheme sets the rendering theme (e.g. "dark", "light", "default", "forest").
func WithTheme(theme string) Option {
	return func(c *Config) {
		c.Theme = theme
	}
}

// WithImageRenderer sets a custom ImageRenderer implementation.
func WithImageRenderer(r ImageRenderer) Option {
	return func(c *Config) {
		c.ImageRenderer = r
	}
}

// WithTextRenderer sets a custom TextRenderer implementation.
func WithTextRenderer(r TextRenderer) Option {
	return func(c *Config) {
		c.TextRenderer = r
	}
}

// WithOnFallback registers a callback for monitoring fallback occurrences.
func WithOnFallback(fn FallbackFunc) Option {
	return func(c *Config) {
		c.OnFallback = fn
	}
}

// WithTerminalEnv sets a custom TerminalEnv for environment lookups.
func WithTerminalEnv(env TerminalEnv) Option {
	return func(c *Config) {
		if env != nil {
			c.TerminalEnv = env
		}
	}
}

// WithColumns sets the column width for text/ASCII diagram layout.
// If <= 0, terminal width is auto-detected.
func WithColumns(cols int) Option {
	return func(c *Config) {
		c.Columns = cols
	}
}

// WithPadding sets horizontal (x) and vertical (y) padding inside node boxes.
func WithPadding(x, y int) Option {
	return func(c *Config) {
		c.PaddingX = x
		c.PaddingY = y
	}
}

// WithSharpEdges enables sharp box corners (┌──┐) instead of rounded corners (╭──╮).
func WithSharpEdges(sharp bool) Option {
	return func(c *Config) {
		c.SharpEdges = sharp
	}
}

// WithHyperlinks enables OSC 8 clickable hyperlinks in terminal text output.
func WithHyperlinks(hyperlinks bool) Option {
	return func(c *Config) {
		c.Hyperlinks = hyperlinks
	}
}

// WithBoxFrame wraps the rendered text diagram in a stylish card frame.
func WithBoxFrame(frame bool) Option {
	return func(c *Config) {
		c.BoxFrame = frame
	}
}

// WithTitle sets an optional diagram title displayed in terminal headers or box frames.
func WithTitle(title string) Option {
	return func(c *Config) {
		c.Title = title
	}
}


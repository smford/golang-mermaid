package mermaid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aaronsb/mmaid-go"
	"golang.org/x/term"
)

// Common errors for text rendering.
var (
	ErrTextParseFailed = errors.New("failed to parse mermaid diagram into text art")
	ErrRasterFailed    = errors.New("failed to rasterize image to ASCII")
)

// ansiRegex matches ANSI escape codes and OSC 8 hyperlinks for calculating visible width.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\a\x1b]*(\a|\x1b\\)`)

// VisibleRuneWidth calculates the terminal character cell count of a string,
// ignoring embedded ANSI color and style escape sequences.
func VisibleRuneWidth(s string) int {
	clean := ansiRegex.ReplaceAllString(s, "")
	return utf8.RuneCountInString(clean)
}

// TextRenderer is the interface for converting Mermaid diagram syntax to ASCII/Unicode text.
type TextRenderer interface {
	RenderText(ctx context.Context, mermaidSource string) (string, error)
}

// MmaidTextRenderer renders Mermaid diagrams as ASCII or Unicode terminal art
// using the pure-Go layout engine in mmaid-go.
type MmaidTextRenderer struct {
	// StrictASCII forces standard 7-bit ASCII (+ - |) instead of Unicode box-drawing.
	StrictASCII bool

	// Theme specifies diagram color/theme (e.g. "default", "slate", "blueprint", "neon", "amber", "phosphor", "monokai").
	Theme string

	// SharpEdges uses sharp corner glyphs (┌──┐) instead of rounded ones (╭──╮).
	SharpEdges bool

	// Hyperlinks enables OSC 8 clickable terminal hyperlinks.
	Hyperlinks bool

	// PaddingX specifies horizontal padding inside node boxes.
	PaddingX int

	// PaddingY specifies vertical padding inside node boxes.
	PaddingY int

	// Columns specifies the layout width in character columns.
	// If <= 0, automatically uses terminal width or a default of 120 columns.
	Columns int

	// BoxFrame wraps the diagram in an outer decorative card border.
	BoxFrame bool

	// Title is an optional title displayed in the frame header.
	Title string
}

// NewMmaidTextRenderer creates a new MmaidTextRenderer with sensible defaults.
func NewMmaidTextRenderer(strictASCII bool, theme string) *MmaidTextRenderer {
	if theme == "" {
		theme = "default"
	}
	return &MmaidTextRenderer{
		StrictASCII: strictASCII,
		Theme:       theme,
		PaddingX:    1,
		PaddingY:    0,
		Columns:     0,
	}
}

// RenderText parses the Mermaid diagram and produces formatted, styled terminal art.
func (m *MmaidTextRenderer) RenderText(ctx context.Context, mermaidSource string) (string, error) {
	mermaidSource = strings.TrimSpace(mermaidSource)
	if mermaidSource == "" {
		return "", ErrEmptySource
	}

	// 1. Determine color support (respect NO_COLOR convention and dumb terminals)
	useColor := !m.StrictASCII && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	themeToUse := m.Theme
	if themeToUse == "" {
		themeToUse = "default"
	}

	// 2. Determine effective terminal column width
	cols := m.Columns
	if cols <= 0 {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 60 {
			cols = w
		} else if cEnv := os.Getenv("COLUMNS"); cEnv != "" {
			if n, err := strconv.Atoi(cEnv); err == nil && n > 60 {
				cols = n
			}
		}
	}
	if cols <= 0 {
		// Default to 120 columns so wide participants and subgraphs don't collide
		cols = 120
	}

	// Set COLUMNS environment variable for mmaid layout engine
	origCols := os.Getenv("COLUMNS")
	_ = os.Setenv("COLUMNS", strconv.Itoa(cols))
	defer func() {
		if origCols != "" {
			_ = os.Setenv("COLUMNS", origCols)
		} else {
			_ = os.Unsetenv("COLUMNS")
		}
	}()

	// 3. Build mmaid options
	var opts []mmaid.Option
	if m.StrictASCII {
		opts = append(opts, mmaid.WithASCII())
	} else if useColor && themeToUse != "" {
		opts = append(opts, mmaid.WithTheme(themeToUse))
	}
	if m.SharpEdges {
		opts = append(opts, mmaid.WithSharpEdges())
	}
	if m.Hyperlinks {
		opts = append(opts, mmaid.WithHyperlinks())
	}
	if m.PaddingX > 0 || m.PaddingY > 0 {
		opts = append(opts, mmaid.WithPadding(m.PaddingX, m.PaddingY))
	}

	result := mmaid.Render(mermaidSource, opts...)
	trimmed := strings.TrimSpace(result)
	if trimmed == "" {
		return "", fmt.Errorf("%w: diagram syntax could not be converted to text art", ErrTextParseFailed)
	}

	// 4. Polish line whitespace and blank lines
	polished := PolishTextDiagram(result)

	// 5. Wrap in outer box frame if requested
	if m.BoxFrame || m.Title != "" {
		polished = WrapInBoxFrame(polished, m.Title, m.SharpEdges, m.StrictASCII, useColor)
	}

	return polished, nil
}

// PolishTextDiagram cleans up raw text diagram output by removing trailing whitespace,
// stripping leading/trailing blank lines, and collapsing excessive consecutive empty lines.
func PolishTextDiagram(raw string) string {
	lines := strings.Split(raw, "\n")
	var cleaned []string
	consecutiveBlanks := 0

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if strings.TrimSpace(trimmed) == "" {
			consecutiveBlanks++
			if consecutiveBlanks <= 1 {
				cleaned = append(cleaned, "")
			}
		} else {
			consecutiveBlanks = 0
			cleaned = append(cleaned, trimmed)
		}
	}

	// Strip leading blank lines
	for len(cleaned) > 0 && cleaned[0] == "" {
		cleaned = cleaned[1:]
	}
	// Strip trailing blank lines
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	if len(cleaned) == 0 {
		return ""
	}

	return strings.Join(cleaned, "\n") + "\n"
}

// WrapInBoxFrame encloses a text diagram in an outer decorative card border.
func WrapInBoxFrame(diagram, title string, sharpEdges, strictASCII, useColor bool) string {
	diagram = strings.TrimSuffix(diagram, "\n")
	lines := strings.Split(diagram, "\n")
	if len(lines) == 0 {
		return diagram
	}

	// Calculate maximum visible content width
	maxContentWidth := 0
	for _, l := range lines {
		w := VisibleRuneWidth(l)
		if w > maxContentWidth {
			maxContentWidth = w
		}
	}

	title = strings.TrimSpace(title)
	titleLen := VisibleRuneWidth(title)
	minWidth := titleLen + 8
	if maxContentWidth < minWidth {
		maxContentWidth = minWidth
	}

	// Inner width: 2 characters padding on left and right
	interiorWidth := maxContentWidth + 4

	// Glyph sets
	var topLeft, topRight, bottomLeft, bottomRight, horiz, vert string
	if strictASCII {
		topLeft, topRight, bottomLeft, bottomRight, horiz, vert = "+", "+", "+", "+", "-", "|"
	} else if sharpEdges {
		topLeft, topRight, bottomLeft, bottomRight, horiz, vert = "┌", "┐", "└", "┘", "─", "│"
	} else {
		topLeft, topRight, bottomLeft, bottomRight, horiz, vert = "╭", "╮", "╰", "╯", "─", "│"
	}

	// Border coloring
	borderPrefix := ""
	borderSuffix := ""
	titlePrefix := ""
	titleSuffix := ""
	if useColor && !strictASCII {
		borderPrefix = "\033[36m"      // Cyan border
		borderSuffix = "\033[0m"
		titlePrefix = "\033[1m\033[37m" // Bold white title
		titleSuffix = "\033[0m"
	}

	var sb strings.Builder

	// Top border
	sb.WriteString(borderPrefix)
	sb.WriteString(topLeft)
	sb.WriteString(horiz)
	sb.WriteString(horiz)
	sb.WriteString(borderSuffix)

	if title != "" {
		sb.WriteString(" ")
		sb.WriteString(titlePrefix)
		sb.WriteString(title)
		sb.WriteString(titleSuffix)
		sb.WriteString(" ")
		sb.WriteString(borderPrefix)
		remaining := interiorWidth - titleLen - 4
		if remaining > 0 {
			sb.WriteString(strings.Repeat(horiz, remaining))
		}
		sb.WriteString(topRight)
		sb.WriteString(borderSuffix)
	} else {
		sb.WriteString(borderPrefix)
		remaining := interiorWidth - 2
		if remaining > 0 {
			sb.WriteString(strings.Repeat(horiz, remaining))
		}
		sb.WriteString(topRight)
		sb.WriteString(borderSuffix)
	}
	sb.WriteString("\n")

	// Empty line at top of frame
	sb.WriteString(borderPrefix)
	sb.WriteString(vert)
	sb.WriteString(borderSuffix)
	sb.WriteString(strings.Repeat(" ", interiorWidth))
	sb.WriteString(borderPrefix)
	sb.WriteString(vert)
	sb.WriteString(borderSuffix)
	sb.WriteString("\n")

	// Content lines with 2 spaces padding
	for _, line := range lines {
		vWidth := VisibleRuneWidth(line)
		paddingRight := maxContentWidth - vWidth
		if paddingRight < 0 {
			paddingRight = 0
		}

		sb.WriteString(borderPrefix)
		sb.WriteString(vert)
		sb.WriteString(borderSuffix)
		sb.WriteString("  ")
		sb.WriteString(line)
		sb.WriteString(strings.Repeat(" ", paddingRight+2))
		sb.WriteString(borderPrefix)
		sb.WriteString(vert)
		sb.WriteString(borderSuffix)
		sb.WriteString("\n")
	}

	// Empty line at bottom of frame
	sb.WriteString(borderPrefix)
	sb.WriteString(vert)
	sb.WriteString(borderSuffix)
	sb.WriteString(strings.Repeat(" ", interiorWidth))
	sb.WriteString(borderPrefix)
	sb.WriteString(vert)
	sb.WriteString(borderSuffix)
	sb.WriteString("\n")

	// Bottom border
	sb.WriteString(borderPrefix)
	sb.WriteString(bottomLeft)
	sb.WriteString(strings.Repeat(horiz, interiorWidth))
	sb.WriteString(bottomRight)
	sb.WriteString(borderSuffix)
	sb.WriteString("\n")

	return sb.String()
}

// RasterASCIIRenderer converts a diagram to an image first, then rasterizes
// the image pixels into terminal ASCII art characters.
type RasterASCIIRenderer struct {
	ImageRenderer ImageRenderer
	Columns       int
}

// NewRasterASCIIRenderer creates a rasterizer using the given ImageRenderer.
// If columns <= 0, 80 columns is used.
func NewRasterASCIIRenderer(imgRenderer ImageRenderer, columns int) *RasterASCIIRenderer {
	if columns <= 0 {
		columns = 80
	}
	return &RasterASCIIRenderer{
		ImageRenderer: imgRenderer,
		Columns:       columns,
	}
}

// RenderText generates PNG bytes via ImageRenderer and converts them to ASCII characters.
func (r *RasterASCIIRenderer) RenderText(ctx context.Context, mermaidSource string) (string, error) {
	if r.ImageRenderer == nil {
		return "", errors.New("no image renderer configured for rasterization")
	}

	imgBytes, err := r.ImageRenderer.RenderImage(ctx, mermaidSource)
	if err != nil {
		return "", fmt.Errorf("rasterize: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", fmt.Errorf("rasterize: decode image: %w", err)
	}

	return ImageToASCII(img, r.Columns), nil
}

// ImageToASCII converts an image.Image to an ASCII art string.
func ImageToASCII(img image.Image, targetWidth int) string {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return ""
	}

	if targetWidth <= 0 {
		targetWidth = 80
	}

	// Terminal characters have an aspect ratio of roughly 1:2 (taller than wide).
	aspect := float64(h) / float64(w)
	targetHeight := int(float64(targetWidth) * aspect * 0.5)
	if targetHeight < 1 {
		targetHeight = 1
	}

	// Ramp from lightest to darkest
	ramp := []rune(" .:-=+*#%@")
	rampLen := len(ramp)

	var buf bytes.Buffer
	for y := 0; y < targetHeight; y++ {
		origY := bounds.Min.Y + int(float64(y)*float64(h)/float64(targetHeight))
		for x := 0; x < targetWidth; x++ {
			origX := bounds.Min.X + int(float64(x)*float64(w)/float64(targetWidth))
			c := img.At(origX, origY)
			gray := color.GrayModel.Convert(c).(color.Gray)

			idx := int((255 - int(gray.Y)) * (rampLen - 1) / 255)
			if idx < 0 {
				idx = 0
			}
			if idx >= rampLen {
				idx = rampLen - 1
			}
			buf.WriteRune(ramp[idx])
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}

// FallbackTextRenderer attempts semantic diagram rendering (mmaid) first.
// If that is unavailable or fails, it attempts rasterization via an ImageRenderer.
// If all text rendering attempts fail, it formats the raw source code in an ASCII box.
type FallbackTextRenderer struct {
	PrimaryRenderer   TextRenderer
	SecondaryRenderer TextRenderer
}

// NewFallbackTextRenderer creates a resilient text renderer with default settings.
func NewFallbackTextRenderer(strictASCII bool, imgRenderer ImageRenderer) *FallbackTextRenderer {
	primary := NewMmaidTextRenderer(strictASCII, "default")
	var secondary TextRenderer
	if imgRenderer != nil {
		secondary = NewRasterASCIIRenderer(imgRenderer, 80)
	}
	return &FallbackTextRenderer{
		PrimaryRenderer:   primary,
		SecondaryRenderer: secondary,
	}
}

// NewFallbackTextRendererFromConfig creates a resilient text renderer populated from a Config struct.
func NewFallbackTextRendererFromConfig(cfg Config) *FallbackTextRenderer {
	strictASCII := cfg.Mode == ModeASCII
	theme := cfg.Theme
	if theme == "" {
		theme = "default"
	}

	primary := &MmaidTextRenderer{
		StrictASCII: strictASCII,
		Theme:       theme,
		SharpEdges:  cfg.SharpEdges,
		Hyperlinks:  cfg.Hyperlinks,
		PaddingX:    cfg.PaddingX,
		PaddingY:    cfg.PaddingY,
		Columns:     cfg.Columns,
		BoxFrame:    cfg.BoxFrame,
		Title:       cfg.Title,
	}

	var secondary TextRenderer
	if cfg.ImageRenderer != nil {
		cols := cfg.Columns
		if cols <= 0 {
			cols = 80
		}
		secondary = NewRasterASCIIRenderer(cfg.ImageRenderer, cols)
	}

	return &FallbackTextRenderer{
		PrimaryRenderer:   primary,
		SecondaryRenderer: secondary,
	}
}

// RenderText executes the fallback chain.
func (f *FallbackTextRenderer) RenderText(ctx context.Context, mermaidSource string) (string, error) {
	if f.PrimaryRenderer != nil {
		out, err := f.PrimaryRenderer.RenderText(ctx, mermaidSource)
		if err == nil && strings.TrimSpace(out) != "" {
			return out, nil
		}
	}

	if f.SecondaryRenderer != nil {
		out, err := f.SecondaryRenderer.RenderText(ctx, mermaidSource)
		if err == nil && strings.TrimSpace(out) != "" {
			return out, nil
		}
	}

	// Final SRE fallback: neatly frame the raw Mermaid source so user still gets the information
	var sb strings.Builder
	sb.WriteString("+----------------------------------------------------------------------+\n")
	sb.WriteString("| [NOTE] ASCII diagram layout engine was unable to parse this diagram.  |\n")
	sb.WriteString("| Raw Mermaid diagram source is displayed below:                       |\n")
	sb.WriteString("+----------------------------------------------------------------------+\n\n")
	sb.WriteString(strings.TrimSpace(mermaidSource))
	sb.WriteString("\n")
	return sb.String(), nil
}

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
	"strings"

	"github.com/aaronsb/mmaid-go"
)

// Common errors for text rendering.
var (
	ErrTextParseFailed = errors.New("failed to parse mermaid diagram into text art")
	ErrRasterFailed    = errors.New("failed to rasterize image to ASCII")
)

// TextRenderer is the interface for converting Mermaid diagram syntax to ASCII/Unicode text.
type TextRenderer interface {
	RenderText(ctx context.Context, mermaidSource string) (string, error)
}

// MmaidTextRenderer renders Mermaid diagrams as ASCII or Unicode terminal art
// using the pure-Go layout engine in mmaid-go.
type MmaidTextRenderer struct {
	// StrictASCII forces standard 7-bit ASCII (+ - |) instead of Unicode box-drawing.
	StrictASCII bool

	// Theme specifies an optional color theme (e.g. "dark", "light", "mono", "neon").
	Theme string

	// SharpEdges uses sharp corner glyphs instead of rounded ones.
	SharpEdges bool
}

// NewMmaidTextRenderer creates a new MmaidTextRenderer.
func NewMmaidTextRenderer(strictASCII bool, theme string) *MmaidTextRenderer {
	return &MmaidTextRenderer{
		StrictASCII: strictASCII,
		Theme:       theme,
	}
}

// RenderText parses the Mermaid diagram and produces terminal art.
func (m *MmaidTextRenderer) RenderText(ctx context.Context, mermaidSource string) (string, error) {
	mermaidSource = strings.TrimSpace(mermaidSource)
	if mermaidSource == "" {
		return "", ErrEmptySource
	}

	var opts []mmaid.Option
	if m.StrictASCII {
		opts = append(opts, mmaid.WithASCII())
	}
	if m.Theme != "" && m.Theme != "default" {
		opts = append(opts, mmaid.WithTheme(m.Theme))
	}
	if m.SharpEdges {
		opts = append(opts, mmaid.WithSharpEdges())
	}

	result := mmaid.Render(mermaidSource, opts...)
	trimmed := strings.TrimSpace(result)
	if trimmed == "" {
		return "", fmt.Errorf("%w: diagram syntax could not be converted to text art", ErrTextParseFailed)
	}

	return result, nil
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

			// Assuming light or dark diagram, map inverted luminance so lines are visible
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

// NewFallbackTextRenderer creates a resilient text renderer.
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

package mermaid

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // register GIF decoder for image conversion
	_ "image/jpeg" // register JPEG decoder for image conversion
	_ "image/png"  // register PNG decoder for image conversion
	"strings"
)

// FormatSixelImage decodes raw image bytes (PNG, JPEG) and encodes them
// into the DEC Sixel bitmap graphics protocol escape sequence.
func FormatSixelImage(imageData []byte, tmuxPassthrough bool) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("sixel: decode image: %w", err)
	}
	return EncodeSixel(img, tmuxPassthrough)
}

// EncodeSixel converts an image.Image into a Sixel graphics escape sequence.
func EncodeSixel(img image.Image, tmuxPassthrough bool) (string, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return "", fmt.Errorf("sixel: empty image bounds")
	}

	// 1. Collect color palette (up to 256 colors)
	colorMap := make(map[color.RGBA]int)
	var palette []color.RGBA

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 128 {
				// Treat transparent background as white
				r, g, b = 0xffff, 0xffff, 0xffff
			}
			c := color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: 255,
			}
			// Quantize slightly if palette grows large
			if len(palette) > 250 {
				c.R = (c.R / 8) * 8
				c.G = (c.G / 8) * 8
				c.B = (c.B / 8) * 8
			}
			if _, exists := colorMap[c]; !exists {
				if len(palette) < 256 {
					colorMap[c] = len(palette)
					palette = append(palette, c)
				}
			}
		}
	}

	// 2. Build Sixel DCS stream
	var sb strings.Builder
	// DCS sequence: \033Pq"1;1;<width>;<height>
	sb.WriteString(fmt.Sprintf("\033Pq\"1;1;%d;%d", w, h))

	// Define palette in Sixel (#index;2;r%;g%;b%)
	for i, c := range palette {
		rPct := int(c.R) * 100 / 255
		gPct := int(c.G) * 100 / 255
		bPct := int(c.B) * 100 / 255
		sb.WriteString(fmt.Sprintf("#%d;2;%d;%d;%d", i, rPct, gPct, bPct))
	}

	// 3. Process 6-pixel vertical bands
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 6 {
		// Collect active colors for this band
		bandColors := make(map[int]bool)
		for dy := 0; dy < 6; dy++ {
			py := y + dy
			if py >= bounds.Max.Y {
				break
			}
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := img.At(x, py).RGBA()
				if a < 128 {
					r, g, b = 0xffff, 0xffff, 0xffff
				}
				c := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
				if len(palette) > 250 {
					c.R = (c.R / 8) * 8
					c.G = (c.G / 8) * 8
					c.B = (c.B / 8) * 8
				}
				if idx, ok := colorMap[c]; ok {
					bandColors[idx] = true
				}
			}
		}

		colorCount := len(bandColors)
		colorIdx := 0
		for cIdx := range bandColors {
			colorIdx++
			sb.WriteString(fmt.Sprintf("#%d", cIdx))

			var runChar byte
			runLen := 0

			flushRun := func() {
				if runLen == 0 {
					return
				}
				if runLen <= 3 {
					for k := 0; k < runLen; k++ {
						sb.WriteByte(runChar)
					}
				} else {
					sb.WriteString(fmt.Sprintf("!%d%c", runLen, runChar))
				}
				runLen = 0
			}

			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				sixBits := byte(0)
				for dy := 0; dy < 6; dy++ {
					py := y + dy
					if py < bounds.Max.Y {
						r, g, b, a := img.At(x, py).RGBA()
						if a < 128 {
							r, g, b = 0xffff, 0xffff, 0xffff
						}
						c := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
						if len(palette) > 250 {
							c.R = (c.R / 8) * 8
							c.G = (c.G / 8) * 8
							c.B = (c.B / 8) * 8
						}
						if curIdx, ok := colorMap[c]; ok && curIdx == cIdx {
							sixBits |= (1 << dy)
						}
					}
				}
				sixelChar := 63 + sixBits
				if runLen > 0 && sixelChar == runChar {
					runLen++
				} else {
					flushRun()
					runChar = sixelChar
					runLen = 1
				}
			}
			flushRun()

			// Return cursor to left of band ($) unless this was the last color in this band
			if colorIdx < colorCount {
				sb.WriteByte('$')
			}
		}

		// Advance to next 6-pixel band (-)
		sb.WriteByte('-')
	}

	// String Terminator
	sb.WriteString("\033\\")

	sixelOutput := sb.String()
	if tmuxPassthrough {
		escaped := strings.ReplaceAll(sixelOutput, "\033", "\033\033")
		return fmt.Sprintf("\033Ptmux;%s\033\\", escaped), nil
	}

	return sixelOutput, nil
}

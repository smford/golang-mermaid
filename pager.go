package mermaid

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Pager provides interactive 2D pan and zoom navigation for large text diagrams.
type Pager struct {
	lines     []string
	maxLineW  int
	rowOffset int
	colOffset int
}

// NewPager creates a new interactive Pager with the given diagram content.
func NewPager(content string) *Pager {
	rawLines := strings.Split(content, "\n")
	maxW := 0
	for _, l := range rawLines {
		w := VisibleRuneWidth(l)
		if w > maxW {
			maxW = w
		}
	}
	return &Pager{
		lines:    rawLines,
		maxLineW: maxW,
	}
}

// Run starts the interactive pager using the provided terminal file descriptors.
func (p *Pager) Run(stdin *os.File, stdout io.Writer) error {
	if stdin == nil || !term.IsTerminal(int(stdin.Fd())) {
		// Non-interactive: write diagram directly
		for _, l := range p.lines {
			if _, err := fmt.Fprintln(stdout, l); err != nil {
				return err
			}
		}
		return nil
	}

	oldState, err := term.MakeRaw(int(stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		_ = term.Restore(int(stdin.Fd()), oldState)
		// Switch back from alternate screen buffer and show cursor
		_, _ = fmt.Fprint(stdout, "\033[?1049l\033[?25h")
	}()

	// Switch to alternate screen buffer (\033[?1049h) and hide cursor (\033[?25l)
	_, _ = fmt.Fprint(stdout, "\033[?1049h\033[?25l")

	buf := make([]byte, 16)
	for {
		termW, termH, err := term.GetSize(int(stdin.Fd()))
		if err != nil || termW <= 0 || termH <= 0 {
			termW, termH = 80, 24
		}
		viewH := termH - 1 // Reserve 1 line for status bar
		if viewH < 1 {
			viewH = 1
		}

		p.draw(stdout, termW, viewH)

		n, err := stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		input := string(buf[:n])

		// Handle user navigation input
		if input == "q" || input == "Q" || input == "\x03" || input == "\x1b" { // q, Ctrl+C, ESC
			break
		} else if input == "j" || input == "\x1b[B" { // Down arrow or j
			if p.rowOffset+1 < len(p.lines) {
				p.rowOffset++
			}
		} else if input == "k" || input == "\x1b[A" { // Up arrow or k
			if p.rowOffset > 0 {
				p.rowOffset--
			}
		} else if input == "l" || input == "\x1b[C" { // Right arrow or l
			p.colOffset += 4
		} else if input == "h" || input == "\x1b[D" { // Left arrow or h
			if p.colOffset >= 4 {
				p.colOffset -= 4
			} else {
				p.colOffset = 0
			}
		} else if input == "g" { // Top
			p.rowOffset = 0
			p.colOffset = 0
		} else if input == "G" { // Bottom
			if len(p.lines) > viewH {
				p.rowOffset = len(p.lines) - viewH
			}
		} else if input == " " || input == "\x1b[6~" { // Page Down / Space
			p.rowOffset += viewH
			if p.rowOffset >= len(p.lines) {
				if len(p.lines) > 0 {
					p.rowOffset = len(p.lines) - 1
				} else {
					p.rowOffset = 0
				}
			}
		} else if input == "\x1b[5~" { // Page Up
			p.rowOffset -= viewH
			if p.rowOffset < 0 {
				p.rowOffset = 0
			}
		} else if input == "r" || input == "R" { // Reset
			p.rowOffset = 0
			p.colOffset = 0
		}
	}

	return nil
}

func (p *Pager) draw(w io.Writer, termW, viewH int) {
	var sb strings.Builder
	// Move cursor to top-left (\033[H) and clear screen (\033[2J)
	sb.WriteString("\033[H\033[2J")

	for r := 0; r < viewH; r++ {
		lineIdx := p.rowOffset + r
		if lineIdx < len(p.lines) {
			line := p.lines[lineIdx]
			sb.WriteString(sliceVisibleString(line, p.colOffset, termW))
		}
		sb.WriteString("\r\n")
	}

	// Inverted status bar (\033[7m ... \033[0m)
	status := fmt.Sprintf(" [Row %d/%d, Col %d]  hjkl / arrows: pan | g/G: top/bottom | r: reset | q: exit ",
		p.rowOffset+1, len(p.lines), p.colOffset)
	if len(status) < termW {
		status += strings.Repeat(" ", termW-len(status))
	} else if len(status) > termW {
		status = status[:termW]
	}
	sb.WriteString("\033[7m" + status + "\033[0m")

	_, _ = io.WriteString(w, sb.String())
}

// sliceVisibleString slices a string horizontally by visible runes.
func sliceVisibleString(s string, colOffset, maxCols int) string {
	runes := []rune(StripANSI(s))
	if colOffset >= len(runes) {
		return ""
	}
	end := colOffset + maxCols
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[colOffset:end])
}

// StripANSI removes ANSI escape codes from a string.
func StripANSI(s string) string {
	var sb strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEsc = false
			}
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

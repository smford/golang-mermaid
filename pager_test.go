package mermaid

import (
	"bytes"
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	colored := "\033[31mRed\033[0m Text and \033[1;34mBold Blue\033[0m"
	stripped := StripANSI(colored)
	expected := "Red Text and Bold Blue"
	if stripped != expected {
		t.Errorf("StripANSI() = %q, expected %q", stripped, expected)
	}
}

func TestSliceVisibleString(t *testing.T) {
	s := "0123456789ABCDEF"
	sliced := sliceVisibleString(s, 5, 5)
	if sliced != "56789" {
		t.Errorf("sliceVisibleString() = %q, expected '56789'", sliced)
	}

	// Offset beyond string length
	empty := sliceVisibleString(s, 20, 5)
	if empty != "" {
		t.Errorf("expected empty string for offset beyond length, got %q", empty)
	}
}

func TestNewPager(t *testing.T) {
	content := "Line 1\nLine 2 is longer\nL3"
	pager := NewPager(content)

	if len(pager.lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(pager.lines))
	}
	if pager.maxLineW != len("Line 2 is longer") {
		t.Errorf("expected maxLineW %d, got %d", len("Line 2 is longer"), pager.maxLineW)
	}
}

func TestPager_Draw(t *testing.T) {
	content := "Line 1: Alpha\nLine 2: Beta\nLine 3: Gamma\nLine 4: Delta"
	pager := NewPager(content)
	pager.rowOffset = 1
	pager.colOffset = 2

	buf := new(bytes.Buffer)
	pager.draw(buf, 40, 2)

	out := buf.String()
	// Should clear screen and move cursor
	if !strings.Contains(out, "\033[H\033[2J") {
		t.Errorf("expected ANSI clear screen and home cursor, got: %q", out)
	}
	// Row offset 1 is "Line 2: Beta", sliced by colOffset 2 -> "ne 2: Beta"
	if !strings.Contains(out, "ne 2: Beta") {
		t.Errorf("expected visible slice of line 2, got: %q", out)
	}
	// Status bar inverted
	if !strings.Contains(out, "\033[7m") || !strings.Contains(out, "Row 2/4") {
		t.Errorf("expected status bar with Row 2/4, got: %q", out)
	}
}

func TestPager_Run_NonTerminal(t *testing.T) {
	content := "Hello\nWorld"
	pager := NewPager(content)
	buf := new(bytes.Buffer)

	// nil stdin simulates non-terminal
	err := pager.Run(nil, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "Hello\nWorld") {
		t.Errorf("expected direct dump of lines in non-terminal mode, got: %q", buf.String())
	}
}

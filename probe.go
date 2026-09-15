package mermaid

import (
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// TerminalCapabilities reports probed hardware/protocol capabilities of a terminal emulator.
type TerminalCapabilities struct {
	SupportsSixel bool
	SupportsKitty bool
	SupportsITerm bool
	RawResponse   string
}

// ParseDA1Response parses a Primary Device Attributes (DA1) response (\033[?<params>c).
// Standard parameter 4 indicates DEC Sixel Graphics support.
func ParseDA1Response(resp string) TerminalCapabilities {
	var caps TerminalCapabilities
	caps.RawResponse = resp

	// Check for DA1: \033[?...c
	start := strings.Index(resp, "\033[?")
	if start == -1 {
		start = strings.Index(resp, "[?")
	}
	if start != -1 {
		offset := start + 3
		if strings.HasPrefix(resp[start:], "[?") {
			offset = start + 2
		}
		end := strings.Index(resp[offset:], "c")
		if end != -1 {
			body := resp[offset : offset+end]
			params := strings.Split(body, ";")
			for _, p := range params {
				p = strings.TrimSpace(p)
				if val, err := strconv.Atoi(p); err == nil {
					if val == 4 {
						// 4 = Sixel Graphics according to DEC STD 070
						caps.SupportsSixel = true
					}
				}
			}
		}
	}

	// Check for Kitty graphics query response: \033_Gi=...
	if strings.Contains(resp, "\033_G") || strings.Contains(resp, "_G") {
		caps.SupportsKitty = true
	}

	return caps
}

// ProbeTerminalStreams queries terminal capabilities via in-band escape sequences.
// It writes the Primary Device Attributes query (\033[c) to out and reads response from in.
func ProbeTerminalStreams(in io.Reader, out io.Writer, timeout time.Duration) TerminalCapabilities {
	if in == nil || out == nil {
		return TerminalCapabilities{}
	}

	// If input is an os.File connected to a terminal, temporarily switch to raw mode so the response isn't line-buffered
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if state, err := term.MakeRaw(int(f.Fd())); err == nil {
			defer func() {
				_ = term.Restore(int(f.Fd()), state)
			}()
		}
	}

	// Send Primary Device Attributes (DA1) query: ESC [ c
	_, err := io.WriteString(out, "\033[c")
	if err != nil {
		return TerminalCapabilities{}
	}

	if timeout <= 0 {
		timeout = 80 * time.Millisecond
	}

	respChan := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		n, readErr := in.Read(buf)
		if readErr == nil && n > 0 {
			respChan <- buf[:n]
		} else {
			respChan <- nil
		}
	}()

	select {
	case data := <-respChan:
		if len(data) == 0 {
			return TerminalCapabilities{}
		}
		return ParseDA1Response(string(data))
	case <-time.After(timeout):
		return TerminalCapabilities{}
	}
}

// ProbeTerminal checks standard terminal capabilities using os.Stdin and os.Stdout.
func ProbeTerminal(timeout time.Duration) TerminalCapabilities {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return TerminalCapabilities{}
	}
	return ProbeTerminalStreams(os.Stdin, os.Stdout, timeout)
}

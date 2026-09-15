package mermaid

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// GraphicsProtocol defines the terminal inline image protocol to use.
type GraphicsProtocol int

const (
	// ProtocolAuto automatically detects the best supported protocol
	// in priority order: Kitty -> iTerm2 -> Sixel -> None (Text).
	ProtocolAuto GraphicsProtocol = iota

	// ProtocolITerm2 uses the iTerm2 OSC 1337 inline image escape code.
	// Supported by: iTerm2, WezTerm, Ghostty, Mintty.
	ProtocolITerm2

	// ProtocolKitty uses the Kitty Graphics Protocol (APC \033_G).
	// Supported by: Kitty, Ghostty, WezTerm.
	ProtocolKitty

	// ProtocolSixel uses the DEC Sixel bitmap graphics protocol (DCS \033Pq).
	// Supported by: Foot, mlterm, Mintty, xterm (with Sixel enabled).
	ProtocolSixel

	// ProtocolNone disables terminal graphics and forces text/ASCII rendering.
	ProtocolNone
)

func (p GraphicsProtocol) String() string {
	switch p {
	case ProtocolAuto:
		return "auto"
	case ProtocolITerm2:
		return "iterm2"
	case ProtocolKitty:
		return "kitty"
	case ProtocolSixel:
		return "sixel"
	case ProtocolNone:
		return "none"
	default:
		return "unknown"
	}
}

// ParseGraphicsProtocol converts a protocol string name into a GraphicsProtocol.
func ParseGraphicsProtocol(s string) (GraphicsProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "":
		return ProtocolAuto, nil
	case "iterm2", "iterm", "osc1337":
		return ProtocolITerm2, nil
	case "kitty":
		return ProtocolKitty, nil
	case "sixel":
		return ProtocolSixel, nil
	case "none", "text", "ascii":
		return ProtocolNone, nil
	default:
		return ProtocolNone, fmt.Errorf("unknown graphics protocol %q (supported: auto, iterm2, kitty, sixel, none)", s)
	}
}

// DetectGraphicsProtocol detects the best available graphics protocol for the given environment.
func DetectGraphicsProtocol(env TerminalEnv, allowCompatible bool) GraphicsProtocol {
	if env == nil {
		env = DefaultTerminalEnv
	}

	termVal := strings.ToLower(env.Getenv("TERM"))
	termProg := strings.ToLower(env.Getenv("TERM_PROGRAM"))

	// 1. Kitty Graphics Protocol
	// Check native Kitty or Ghostty
	if env.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(termVal, "kitty") {
		return ProtocolKitty
	}
	if termProg == "ghostty" {
		return ProtocolKitty
	}

	// 2. iTerm2 Protocol
	if IsITerm2Env(env) {
		return ProtocolITerm2
	}
	if allowCompatible {
		if termProg == "wezterm" || termProg == "mintty" {
			return ProtocolITerm2
		}
	}

	// 3. DEC Sixel Protocol
	if env.Getenv("SIXEL_SUPPORT") == "1" ||
		termVal == "foot" ||
		termVal == "mlterm" ||
		strings.Contains(termVal, "sixel") {
		return ProtocolSixel
	}

	return ProtocolNone
}

// SupportsGraphicsProtocol checks if the given protocol is supported in the specified environment.
func SupportsGraphicsProtocol(env TerminalEnv, proto GraphicsProtocol, allowCompatible bool) bool {
	if env == nil {
		env = DefaultTerminalEnv
	}
	switch proto {
	case ProtocolAuto:
		return DetectGraphicsProtocol(env, allowCompatible) != ProtocolNone
	case ProtocolKitty:
		termVal := strings.ToLower(env.Getenv("TERM"))
		termProg := strings.ToLower(env.Getenv("TERM_PROGRAM"))
		if env.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(termVal, "kitty") || termProg == "ghostty" {
			return true
		}
		if allowCompatible && termProg == "wezterm" {
			return true
		}
		return false
	case ProtocolITerm2:
		return SupportsITerm2ImagesEnv(env, allowCompatible)
	case ProtocolSixel:
		termVal := strings.ToLower(env.Getenv("TERM"))
		return env.Getenv("SIXEL_SUPPORT") == "1" ||
			termVal == "foot" ||
			termVal == "mlterm" ||
			strings.Contains(termVal, "sixel")
	case ProtocolNone:
		return true
	default:
		return false
	}
}


// FormatKittyImage formats raw image data using the Kitty graphics protocol (APC \033_G).
func FormatKittyImage(pngData []byte, widthCols, heightRows int, tmuxPassthrough bool) string {
	encoded := base64.StdEncoding.EncodeToString(pngData)
	const chunkSize = 4096
	var sb strings.Builder

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		m := 1
		if end >= len(encoded) {
			end = len(encoded)
			m = 0
		}
		chunk := encoded[i:end]

		var cmd string
		if i == 0 {
			var ctrl []string
			ctrl = append(ctrl, "a=T", "f=100") // action=Transmit&display, format=PNG
			if m == 1 {
				ctrl = append(ctrl, "m=1")
			} else {
				ctrl = append(ctrl, "m=0")
			}
			if widthCols > 0 {
				ctrl = append(ctrl, fmt.Sprintf("c=%d", widthCols))
			}
			if heightRows > 0 {
				ctrl = append(ctrl, fmt.Sprintf("r=%d", heightRows))
			}
			ctrl = append(ctrl, "q=2") // quiet mode: suppress terminal ok response
			cmd = fmt.Sprintf("\033_G%s;%s\033\\", strings.Join(ctrl, ","), chunk)
		} else {
			cmd = fmt.Sprintf("\033_Gm=%d;%s\033\\", m, chunk)
		}

		if tmuxPassthrough {
			escaped := strings.ReplaceAll(cmd, "\033", "\033\033")
			sb.WriteString(fmt.Sprintf("\033Ptmux;%s\033\\", escaped))
		} else {
			sb.WriteString(cmd)
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// FormatTerminalImage formats raw image data according to the target GraphicsProtocol.
func FormatTerminalImage(proto GraphicsProtocol, imgData []byte, opts ImageFormatOptions) (string, error) {
	switch proto {
	case ProtocolKitty:
		widthCols := 0
		if strings.HasSuffix(opts.Width, "cell") {
			if n, err := strconv.Atoi(strings.TrimSuffix(opts.Width, "cell")); err == nil {
				widthCols = n
			}
		}
		heightRows := 0
		if strings.HasSuffix(opts.Height, "cell") {
			if n, err := strconv.Atoi(strings.TrimSuffix(opts.Height, "cell")); err == nil {
				heightRows = n
			}
		}
		return FormatKittyImage(imgData, widthCols, heightRows, opts.TmuxPassthrough), nil

	case ProtocolSixel:
		return FormatSixelImage(imgData, opts.TmuxPassthrough)

	case ProtocolITerm2:
		return FormatITerm2Image(imgData, opts), nil

	default:
		return "", fmt.Errorf("unsupported graphics protocol: %v", proto)
	}
}

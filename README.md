# golang-mermaid

[![Go Reference](https://pkg.go.dev/badge/github.com/smford/golang-mermaid.svg)](https://pkg.go.dev/github.com/smford/golang-mermaid)
[![Go Report Card](https://goreportcard.com/badge/github.com/smford/golang-mermaid)](https://goreportcard.com/report/github.com/smford/golang-mermaid)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**`golang-mermaid`** is a production-grade Go module designed to render and display Mermaid diagrams directly inside the terminal.

It leverages the **iTerm2 Inline Image Protocol (`OSC 1337`)** to render crisp, high-resolution graphical diagrams in iTerm2 and compatible modern terminal emulators. When running in a standard terminal (such as Apple Terminal, basic xterm, Linux TTY, or CI/CD pipelines), when output is piped to a file, or if image rendering services are unreachable, it **gracefully degrades** to clean, high-fidelity ASCII or Unicode box-drawing terminal art.

---

## Architecture & Reliability

Engineered with **Site Reliability Engineering (SRE)** and senior development principles:

```mermaid
flowchart TD
    Start[Input: Mermaid File or String] --> Detect{Detect Terminal & TTY}
    
    Detect -- "iTerm2 / WezTerm & Interactive TTY" --> TryImage[Attempt Image Rendering]
    Detect -- "Standard Terminal / Pipe / File" --> FallbackTrigger[Trigger SRE Fallback Hook]
    
    subgraph ImagePipeline [Resilient Image Pipeline]
        TryImage --> LocalCLI{Local mmdc installed?}
        LocalCLI -- Yes --> ExecLocal[Render via local mmdc CLI]
        LocalCLI -- No --> RemoteKroki[POST to Kroki API]
        RemoteKroki -- Timeout / Error --> RemoteInk[GET via Mermaid.ink]
    end
    
    ExecLocal -- Success --> FormatOSC[Format iTerm2 OSC 1337 Sequence]
    RemoteKroki -- Success --> FormatOSC
    RemoteInk -- Success --> FormatOSC
    
    RemoteInk -- Failure / All Fail --> FallbackTrigger
    
    subgraph TextPipeline [Text Fallback Pipeline]
        FallbackTrigger --> SemanticEngine{mmaid Layout Engine}
        SemanticEngine -- Success --> RenderUnicode[Render Unicode / ASCII Box Diagram]
        SemanticEngine -- Unsupported Syntax --> RasterEngine[Rasterize Image to ASCII]
        RasterEngine -- Failure --> SourceFraming[Neatly Frame Raw Mermaid Source]
    end
    
    FormatOSC --> Output[Terminal Output Stream]
    RenderUnicode --> Output
    RasterEngine --> Output
    SourceFraming --> Output
```

### Key Engineering Highlights

- **Zero-Panic Resiliency**: Strict error propagation; never panics on malformed syntax or terminal anomalies.
- **Cascading Fallback**: Automatically cascades through local CLI (`mmdc`), remote REST services (Kroki, Mermaid.ink), and local pure-Go layout engines (`mmaid-go`).
- **Deadline & Timeout Budgets**: All network operations respect `context.Context` deadlines. The multi-backend image renderer automatically budgets time slices across candidates to prevent one sluggish service from exhausting the entire deadline.
- **SRE Observability Hooks**: Register custom `OnFallback` callbacks to log warnings, emit Prometheus metrics, or track terminal degradation across developer fleets.
- **TTY & Pipe Safety**: Employs interactive terminal checks via `golang.org/x/term` so that binary escape sequences are never accidentally spewed into redirected log files or shell pipes unless explicitly forced.
- **Tmux Passthrough**: Automatically detects tmux sessions and wraps escape sequences with DCS passthrough (`\033Ptmux;...\033\\`).

---

## Installation

```bash
go get github.com/smford/golang-mermaid
```

Requires Go 1.21 or later.

---

## Quick Start

### 1. Print a Diagram String

```go
package main

import (
    "log"
    mermaid "github.com/smford/golang-mermaid"
)

func main() {
    chart := `
    graph TD
        Client[Client App] --> LB[Load Balancer]
        LB --> Web1[Web Server 1]
        LB --> Web2[Web Server 2]
    `

    if err := mermaid.Print(chart); err != nil {
        log.Fatalf("Failed to print diagram: %v", err)
    }
}
```

### 2. Print a Mermaid File from Disk

```go
package main

import (
    "log"
    mermaid "github.com/smford/golang-mermaid"
)

func main() {
    if err := mermaid.PrintFile("testdata/architecture.mmd"); err != nil {
        log.Fatalf("Failed to render diagram: %v", err)
    }
}
```

---

## Configuration & Options

`golang-mermaid` uses the functional options pattern for complete flexibility:

```go
printer := mermaid.New(
    // Rendering mode: ModeAuto (default), ModeImage, ModeASCII, ModeUnicode
    mermaid.WithMode(mermaid.ModeAuto),

    // iTerm2 display dimensions
    mermaid.WithWidth("80%"),
    mermaid.WithHeight("auto"),
    mermaid.WithPreserveAspectRatio(true),

    // Execution timeout for external renderers
    mermaid.WithTimeout(8 * time.Second),

    // Output destination (defaults to os.Stdout)
    mermaid.WithWriter(os.Stdout),

    // SRE Observability callback
    mermaid.WithOnFallback(func(reason string, err error) {
        log.Printf("[OBSERVABILITY] Degraded to ASCII diagram: %s", reason)
    }),
)

err := printer.PrintFile("testdata/incident_response.mmd")
```

### Supported Options

| Option | Description | Default |
| :--- | :--- | :--- |
| `WithMode(mode)` | Set render mode (`ModeAuto`, `ModeImage`, `ModeASCII`, `ModeUnicode`) | `ModeAuto` |
| `WithWidth(width)` | Set display width in iTerm2 (`"auto"`, `"80%"`, `"800px"`, `"60cell"`) | `"auto"` |
| `WithHeight(height)` | Set display height in iTerm2 (`"auto"`, `"400px"`, `"30cell"`) | `"auto"` |
| `WithPreserveAspectRatio(bool)` | Maintain image aspect ratio in iTerm2 | `true` |
| `WithTimeout(duration)` | Maximum time budget for rendering requests | `10s` |
| `WithWriter(w)` | Destination `io.Writer` | `os.Stdout` |
| `WithOnFallback(fn)` | Callback executed whenever fallback from image to text occurs | `nil` |
| `WithAllowCompatibleTerminals(bool)` | Allow terminals that implement OSC 1337 (e.g. WezTerm, Ghostty) | `true` |
| `WithForceTTY(bool)` | Bypass TTY detection (useful in automated tests or pseudo-terminals) | `false` |
| `WithDisableFallback(bool)` | Fail immediately with error instead of falling back to ASCII | `false` |
| `WithImageRenderer(r)` | Supply a custom implementation of `ImageRenderer` | Resilient chain |
| `WithTextRenderer(r)` | Supply a custom implementation of `TextRenderer` | Fallback text |

---

## In-Memory Rendering

If you want the formatted output string or raw image bytes without printing directly to a terminal stream:

```go
ctx := context.Background()
res, err := mermaid.Render(ctx, diagramSource, mermaid.WithMode(mermaid.ModeAuto))
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Rendered in: %v\n", res.Duration)
fmt.Printf("Actual Mode: %s\n", res.Mode)
fmt.Printf("Fallback Occurred: %v (Reason: %s)\n", res.FallbackOccurred, res.FallbackReason)

if res.Mode == mermaid.ModeImage {
    fmt.Printf("Received %d bytes of PNG image data\n", len(res.ImageData))
}

// Print formatted terminal sequence or text art:
fmt.Print(res.Output)
```

---

## Working Example & CLI Application

The repository includes both a simple example and a production CLI utility:

### Simple Example

```bash
go run examples/simple/main.go testdata/architecture.mmd
```

### CLI Tool (`mermaid-term`)

Build and run the full-featured CLI:

```bash
# Build
go build -o bin/mermaid-term cmd/mermaid-term/main.go

# Auto-detect (iTerm2 image or ASCII)
./bin/mermaid-term testdata/architecture.mmd

# Force ASCII mode
./bin/mermaid-term -mode=ascii testdata/incident_response.mmd

# Force Unicode mode
./bin/mermaid-term -mode=unicode testdata/sequence_auth.mmd

# Enable verbose SRE diagnostic logs
./bin/mermaid-term -v testdata/state_machine.mmd

# Read diagram from stdin
cat testdata/database_er.mmd | ./bin/mermaid-term
```

---

## Test Diagrams Included

Five test diagrams are provided under [`testdata/`](testdata/):

1. **`architecture.mmd`**: Microservices topology with CloudFront, ALB, VPC, Kafka, Redis, and PostgreSQL.
2. **`incident_response.mmd`**: SRE incident response runbook and escalation flowchart.
3. **`sequence_auth.mmd`**: Zero-trust authentication sequence with Okta SSO, WebAuthn MFA, and HashiCorp Vault.
4. **`state_machine.mmd`**: Kubernetes Pod lifecycle state transitions (`CrashLoopBackOff`, `Running`, etc.).
5. **`database_er.mmd`**: Relational data model for Service Level Objectives (SLOs) and incident tracking.

---

## How the iTerm2 Inline Image Protocol Works

Under the hood, iTerm2 supports an escape sequence defined by the OSC 1337 specification:

```
ESC ] 1337 ; File = [args] : <base64-encoded image payload> ^G
```

- `ESC` is ASCII 27 (`\033`)
- `^G` is ASCII 7 (`BEL` or `\a`)
- Arguments include `inline=1` (renders the image in the scrollback buffer instead of saving to disk), `width=<value>`, `height=<value>`, and `preserveAspectRatio=1`.

When running in tmux, the escape code is wrapped in tmux's Device Control String (DCS) passthrough:

```
ESC P tmux ; ESC ESC ] 1337 ; File = ... ^G ESC \
```

`golang-mermaid` handles this encoding, argument formatting, and tmux passthrough automatically.

---

## Terminal Compatibility

| Terminal Emulator | ModeAuto Result | Notes |
| :--- | :--- | :--- |
| **iTerm2** | 🖼️ Inline PNG Image | Native OSC 1337 support |
| **WezTerm** | 🖼️ Inline PNG Image | Full OSC 1337 implementation |
| **Ghostty** | 🖼️ Inline PNG Image | Native OSC 1337 support |
| **mintty** | 🖼️ Inline PNG Image | Windows terminal with OSC 1337 |
| **Apple Terminal** | 🔤 Unicode Box Art | Graceful fallback (no image support) |
| **Linux VT / Alacritty** | 🔤 Unicode Box Art | Graceful fallback (no image support) |
| **CI/CD Runners (GitHub Actions)** | 🔤 Unicode / ASCII | Graceful fallback (non-interactive TTY) |
| **Output piped to file (`> out.txt`)** | 🔤 Unicode / ASCII | Safe TTY detection protects output files |

---

## Testing

Run the test suite with the Go race detector:

```bash
go test -v -race ./...
```

All unit tests and diagram validation tests run with 100% pass rate.

---

## License

MIT License. See [LICENSE](LICENSE) for details.

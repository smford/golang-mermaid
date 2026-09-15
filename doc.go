// Package mermaid provides resilient, terminal-native rendering for Mermaid diagrams.
//
// It targets modern terminal emulators by supporting multiple inline graphics protocols:
//   - Kitty Graphics Protocol (APC \033_G) for Kitty, Ghostty, and WezTerm.
//   - iTerm2 Inline Image Protocol (OSC 1337) for iTerm2, WezTerm, Ghostty, and Mintty.
//   - DEC Sixel Bitmap Protocol (DCS \033Pq) for Foot, mlterm, Mintty, and Sixel-enabled terminals.
//
// When running in terminals that do not support graphics protocols, when output is piped
// or redirected, or when image generation fails, the package gracefully degrades to rendering
// clean, high-fidelity ASCII or Unicode box-drawing diagrams directly in the console.
//
// # Architecture and Reliability
//
// Designed with Site Reliability Engineering (SRE) principles in mind:
//
//   - Graceful Degradation: High-fidelity graphics are preferred in capable terminals,
//     with seamless fallback to Unicode or 7-bit ASCII text diagrams when inline images
//     are unavailable.
//   - Multi-Protocol Graphics: Automatically negotiates Kitty APC, iTerm2 OSC, or DEC Sixel DCS
//     with transparent tmux passthrough support.
//   - Content-Addressed Caching: Thread-safe in-memory and atomic filesystem caching
//     indexed by SHA-256 hashes of diagram source and render options.
//   - Air-Gapped & Offline Operation: Explicit offline mode avoids external HTTP requests,
//     leveraging local mmdc CLI if present or immediately falling back to Unicode text art.
//   - Markdown Runbook Extraction: Parses Markdown operational documents, replacing
//     embedded ```mermaid blocks inline with rendered terminal output while preserving text.
//   - In-Band PTY Capability Probing: Sends Primary Device Attributes queries (\033[c)
//     to identify terminal capabilities over SSH sessions where TERM_PROGRAM is stripped.
//   - Interactive 2D Pan & Zoom Pager: Renders diagrams inside an alternate screen buffer
//     with 2D navigation (arrow keys, vi keys hjkl, page jumps, reset, exit).
//   - Production Telemetry & Observability: Built-in Prometheus metrics exposition,
//     thread-safe stats accumulators, and SRE fallback hooks.
//   - Safe I/O: Automatically detects interactive TTYs to prevent spewing binary escape
//     codes into pipes or redirected log files.
//
// # Quick Start
//
// Print a Mermaid diagram string:
//
//	err := mermaid.Print(`
//	    graph TD
//	        Client --> LoadBalancer
//	        LoadBalancer --> BackendService
//	`)
//
// Print a Mermaid diagram file from disk:
//
//	err := mermaid.PrintFile("architecture.mmd")
//
// Render with Content Caching and Telemetry:
//
//	stats := mermaid.NewStatsRecorder()
//	printer := mermaid.New(
//	    mermaid.WithCache(true),
//	    mermaid.WithTelemetry(stats),
//	)
//	err := printer.PrintFile("architecture.mmd")
//	fmt.Println(stats.ExportPrometheus())
//
// Render Markdown Runbook:
//
//	printer := mermaid.New(mermaid.WithMode(mermaid.ModeAuto))
//	err := printer.PrintMarkdownFile(context.Background(), "runbook.md")
//
// Interactive Pager for Large Diagrams:
//
//	err := mermaid.PrintFile("architecture.mmd", mermaid.WithInteractive(true))
//
// Air-Gapped / Offline Mode:
//
//	err := mermaid.PrintFile("architecture.mmd", mermaid.WithOffline(true))
//
// In-Band Terminal Probing (SSH):
//
//	err := mermaid.PrintFile("architecture.mmd", mermaid.WithTerminalProbe(true))
package mermaid

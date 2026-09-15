// Package mermaid provides resilient, terminal-native rendering for Mermaid diagrams.
//
// It natively targets iTerm2 terminals by leveraging the iTerm2 inline image display
// protocol (OSC 1337). When running in terminals that do not support inline images,
// when output is piped or redirected, or when image generation fails, the package
// gracefully degrades to rendering clean, high-fidelity ASCII or Unicode box-drawing
// diagrams directly in the console.
//
// # Architecture and Reliability
//
// Designed with Site Reliability Engineering (SRE) principles in mind:
//
//   - Graceful Degradation: High-fidelity graphics are preferred in capable terminals,
//     with seamless fallback to Unicode or 7-bit ASCII text diagrams when inline images
//     are unavailable.
//   - Context & Timeout Budgets: All rendering operations accept context.Context and
//     enforce strict network/CLI timeouts to prevent hanging processes.
//   - Zero Unhandled Failures: Fallback cascades from local binaries (mmdc) to remote
//     HTTP renderers (Kroki, Mermaid.ink) to local pure-Go layout engines (mmaid-go),
//     finally falling back to framed source code in extreme cases.
//   - Observability: Diagnostic hooks (OnFallback) allow callers to log or emit metrics
//     whenever fallbacks occur.
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
// Force ASCII mode:
//
//	err := mermaid.PrintFile("architecture.mmd", mermaid.WithMode(mermaid.ModeASCII))
//
// Custom Observability Hook:
//
//	err := mermaid.PrintFile("architecture.mmd",
//	    mermaid.WithOnFallback(func(reason string, err error) {
//	        log.Printf("[WARN] Degraded to ASCII diagram: %s", reason)
//	    }),
//	)
package mermaid

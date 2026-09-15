package main

import (
	"context"
	"fmt"
	"log"
	"os"

	mermaid "github.com/smford/golang-mermaid"
)

func main() {
	// Default to our production architecture diagram if no file is specified
	targetFile := "testdata/architecture.mmd"
	if len(os.Args) > 1 {
		targetFile = os.Args[1]
	}

	fmt.Printf("==> golang-mermaid SRE Diagram Runner: %s\n\n", targetFile)

	// SRE Observability: register a telemetry metrics recorder
	stats := mermaid.NewStatsRecorder()

	// SRE Observability: register a fallback notification hook
	fallbackHook := func(reason string, err error) {
		log.Printf("[Observability] Image fallback triggered: %s", reason)
	}

	// Create a printer with multi-protocol auto-detection, content caching,
	// terminal probing, styling, and SRE telemetry
	printer := mermaid.New(
		mermaid.WithMode(mermaid.ModeAuto),
		mermaid.WithGraphicsProtocol(mermaid.ProtocolAuto),
		mermaid.WithWidth("80%"),
		mermaid.WithTheme("default"),
		mermaid.WithBoxFrame(true),
		mermaid.WithTitle("Mermaid Diagram"),
		mermaid.WithCache(true),
		mermaid.WithTerminalProbe(true),
		mermaid.WithTelemetry(stats),
		mermaid.WithOnFallback(fallbackHook),
	)

	ctx := context.Background()

	// Automatically detect and render Markdown runbooks or standard .mmd files
	var err error
	if mermaid.IsMarkdownFile(targetFile) {
		err = printer.PrintMarkdownFile(ctx, targetFile)
	} else {
		err = printer.PrintFileContext(ctx, targetFile)
	}

	if err != nil {
		log.Fatalf("Error rendering diagram: %v", err)
	}

	// Display SRE telemetry metrics summary
	fmt.Printf("\n--- SRE Telemetry Summary ---\n")
	fmt.Printf("Total Renders:   %d\n", stats.TotalRenders)
	fmt.Printf("Cache Hits:      %d\n", stats.CacheHits)
	fmt.Printf("Fallbacks:       %d\n", stats.Fallbacks)
	fmt.Printf("Average Latency: %v\n", stats.AverageDuration())
}

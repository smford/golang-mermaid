package main

import (
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

	fmt.Printf("==> Rendering Mermaid Diagram: %s\n", targetFile)

	// SRE Observability: register a fallback notification hook
	fallbackHook := func(reason string, err error) {
		log.Printf("[Observability] Fallback triggered: %s", reason)
	}

	// Create a printer with auto-detection, ANSI color themes, styling and fallback monitoring
	printer := mermaid.New(
		mermaid.WithMode(mermaid.ModeAuto),
		mermaid.WithWidth("80%"),
		mermaid.WithTheme("default"),
		mermaid.WithBoxFrame(true),
		mermaid.WithTitle("Mermaid Architecture Diagram"),
		mermaid.WithOnFallback(fallbackHook),
	)

	// Print the diagram to stdout
	if err := printer.PrintFile(targetFile); err != nil {
		log.Fatalf("Error rendering diagram: %v", err)
	}
}

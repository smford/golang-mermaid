package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	mermaid "github.com/smford/golang-mermaid"
)

func main() {
	var (
		filePath    = flag.String("file", "", "Path to Mermaid diagram file (.mmd or .mermaid). If omitted, reads from stdin.")
		modeStr     = flag.String("mode", "auto", "Rendering mode: 'auto', 'image', 'ascii', or 'unicode'")
		protocolStr = flag.String("protocol", "auto", "Terminal graphics protocol: 'auto', 'kitty', 'iterm2', 'sixel', or 'none'")
		width       = flag.String("width", "auto", "Image width for iTerm2/Kitty (e.g. 'auto', '80%', '800px', '60cell')")
		height      = flag.String("height", "auto", "Image height for iTerm2/Kitty (e.g. 'auto', '400px', '30cell')")
		krokiURL    = flag.String("kroki-url", "https://kroki.io", "Base URL for the Kroki diagram rendering service")
		theme       = flag.String("theme", "default", "Diagram theme ('default', 'slate', 'blueprint', 'neon', 'amber', 'phosphor', 'monokai')")
		columns     = flag.Int("columns", 0, "Terminal column width override for text diagram layout (0 for auto)")
		paddingX    = flag.Int("padding-x", 1, "Horizontal box padding inside nodes")
		paddingY    = flag.Int("padding-y", 0, "Vertical box padding inside nodes")
		sharp       = flag.Bool("sharp", false, "Use sharp corners (┌──┐) instead of rounded corners (╭──╮)")
		hyperlinks  = flag.Bool("hyperlinks", false, "Enable OSC 8 terminal hyperlinks for click actions")
		frame       = flag.Bool("frame", false, "Wrap diagram in an executive card border frame")
		title       = flag.String("title", "", "Optional diagram title displayed in frame header")
		noColor     = flag.Bool("no-color", false, "Disable ANSI color output in text diagrams")
		timeoutSec  = flag.Int("timeout", 10, "Timeout in seconds for remote/CLI rendering operations")
		verbose     = flag.Bool("v", false, "Enable verbose SRE logging (diagnostics, fallback reasons, duration)")
		forceTTY    = flag.Bool("force-tty", false, "Force treating output as an interactive TTY")
		scale       = flag.Float64("scale", 1.0, "Rasterization scale factor for image rendering (e.g. 1.0, 2.0 for Retina/HiDPI)")
		cache       = flag.Bool("cache", true, "Enable content-addressed diagram disk caching")
		cacheDir    = flag.String("cache-dir", "", "Custom directory for disk cache (default: ~/.cache/golang-mermaid)")
		cacheTTL    = flag.Duration("cache-ttl", 24*time.Hour, "Time-to-live for cached diagram renders (e.g. 24h, 30m)")
		clearCache  = flag.Bool("clear-cache", false, "Clear all cached diagrams and exit")
		offline     = flag.Bool("offline", false, "Enforce air-gapped/offline mode (disable all remote HTTP renderers)")
		markdown    = flag.Bool("markdown", false, "Parse Markdown document and render embedded ```mermaid blocks inline")
		probe       = flag.Bool("probe", false, "Probe terminal capabilities via in-band PTY escape queries (ideal for SSH sessions)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mermaid-term [options] [file.mmd]\n\n")
		fmt.Fprintf(os.Stderr, "Prints Mermaid diagrams in terminals using Kitty, iTerm2, or Sixel graphics, with ASCII fallback.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  mermaid-term testdata/architecture.mmd\n")
		fmt.Fprintf(os.Stderr, "  mermaid-term -protocol=kitty testdata/sequence_auth.mmd\n")
		fmt.Fprintf(os.Stderr, "  mermaid-term -protocol=sixel testdata/state_machine.mmd\n")
		fmt.Fprintf(os.Stderr, "  mermaid-term -mode=ascii testdata/incident_response.mmd\n")
		fmt.Fprintf(os.Stderr, "  mermaid-term -width=80%% testdata/sequence_auth.mmd\n")
		fmt.Fprintf(os.Stderr, "  cat testdata/state_machine.mmd | mermaid-term\n")
	}

	flag.Parse()

	// Clear cache command
	if *clearCache {
		c, err := mermaid.NewDiskCache(*cacheDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing cache: %v\n", err)
			os.Exit(1)
		}
		if err := c.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("golang-mermaid: cache cleared successfully.")
		return
	}

	// Positional argument fallback for file path
	targetFile := *filePath
	if targetFile == "" && flag.NArg() > 0 {
		targetFile = flag.Arg(0)
	}

	// Read input: file or stdin
	var diagramSource string
	if targetFile != "" {
		data, err := os.ReadFile(targetFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read file %q: %v\n", targetFile, err)
			os.Exit(1)
		}
		diagramSource = string(data)
	} else {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read from stdin: %v\n", err)
			os.Exit(1)
		}
		diagramSource = string(data)
		if strings.TrimSpace(diagramSource) == "" {
			flag.Usage()
			os.Exit(1)
		}
	}

	// Parse mode
	var mode mermaid.RenderMode
	switch strings.ToLower(*modeStr) {
	case "auto":
		mode = mermaid.ModeAuto
	case "image":
		mode = mermaid.ModeImage
	case "ascii":
		mode = mermaid.ModeASCII
	case "unicode":
		mode = mermaid.ModeUnicode
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown mode %q (must be 'auto', 'image', 'ascii', or 'unicode')\n", *modeStr)
		os.Exit(1)
	}

	// Parse graphics protocol
	proto, err := mermaid.ParseGraphicsProtocol(*protocolStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	timeout := time.Duration(*timeoutSec) * time.Second

	// SRE Observability logging
	fallbackHook := func(reason string, err error) {
		if *verbose {
			fmt.Fprintf(os.Stderr, "[SRE Info] Fallback occurred: %s\n", reason)
		}
	}

	if *noColor {
		_ = os.Setenv("NO_COLOR", "1")
	}

	// Build printer options
	printerOpts := []mermaid.Option{
		mermaid.WithMode(mode),
		mermaid.WithGraphicsProtocol(proto),
		mermaid.WithWidth(*width),
		mermaid.WithHeight(*height),
		mermaid.WithTheme(*theme),
		mermaid.WithColumns(*columns),
		mermaid.WithPadding(*paddingX, *paddingY),
		mermaid.WithSharpEdges(*sharp),
		mermaid.WithHyperlinks(*hyperlinks),
		mermaid.WithBoxFrame(*frame),
		mermaid.WithTitle(*title),
		mermaid.WithTimeout(timeout),
		mermaid.WithScale(*scale),
		mermaid.WithForceTTY(*forceTTY),
		mermaid.WithOnFallback(fallbackHook),
		mermaid.WithCache(*cache),
		mermaid.WithCacheTTL(*cacheTTL),
		mermaid.WithTerminalProbe(*probe),
	}
	if *offline {
		printerOpts = append(printerOpts, mermaid.WithOffline(true))
	} else {
		printerOpts = append(printerOpts, mermaid.WithImageRenderer(mermaid.NewResilientImageRendererWithScale(*krokiURL, timeout, *scale)))
	}
	if *cacheDir != "" {
		printerOpts = append(printerOpts, mermaid.WithCacheDir(*cacheDir))
	}

	printer := mermaid.New(printerOpts...)

	if *verbose {
		fmt.Fprintf(os.Stderr, "[SRE Info] Terminal iTerm2: %v | Detected Protocol: %s | Destination TTY: %v\n",
			mermaid.IsITerm2(),
			mermaid.DetectGraphicsProtocol(nil, true),
			mermaid.IsTerminal(os.Stdout),
		)
		fmt.Fprintf(os.Stderr, "[SRE Info] Configured Mode: %s | Protocol: %s | Cache: %v | Width: %s | Timeout: %v\n", mode, proto, *cache, *width, timeout)
	}

	ctx := context.Background()

	// Check if input is a Markdown file or explicitly requested via -markdown
	if *markdown || (targetFile != "" && mermaid.IsMarkdownFile(targetFile)) {
		renderedMD, err := printer.RenderMarkdown(ctx, diagramSource)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering markdown: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(renderedMD)
		return
	}

	res, err := printer.Render(ctx, diagramSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering diagram: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "[SRE Info] Rendered in %v using mode: %s (protocol: %s, cache hit: %v, fallback: %v)\n\n",
			res.Duration, res.Mode, res.Protocol, res.CacheHit, res.FallbackOccurred)
	}

	// Output result
	fmt.Print(res.Output)
	if !strings.HasSuffix(res.Output, "\n") {
		fmt.Println()
	}
}

package mermaid

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RenderMarkdown parses markdown text, replaces all ```mermaid code blocks
// with their rendered terminal representations (inline graphics or ASCII/Unicode art),
// and returns the resulting document. Surrounding Markdown content is preserved.
func (p *Printer) RenderMarkdown(ctx context.Context, markdown string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	var out strings.Builder

	var inMermaidBlock bool
	var mermaidFence string
	var mermaidLines []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inMermaidBlock {
			if strings.HasPrefix(trimmed, "```mermaid") || strings.HasPrefix(trimmed, "~~~mermaid") {
				inMermaidBlock = true
				if strings.HasPrefix(trimmed, "```") {
					mermaidFence = "```"
				} else {
					mermaidFence = "~~~"
				}
				mermaidLines = nil
				continue
			}
			out.WriteString(line)
			out.WriteString("\n")
		} else {
			if strings.HasPrefix(trimmed, mermaidFence) {
				inMermaidBlock = false
				mermaidSource := strings.Join(mermaidLines, "\n")

				res, err := p.Render(ctx, mermaidSource)
				if err != nil {
					if p.config.DisableFallback {
						return "", fmt.Errorf("render markdown mermaid block: %w", err)
					}
					// If fallback disabled is false, preserve raw block on error
					out.WriteString(mermaidFence + "mermaid\n")
					out.WriteString(mermaidSource + "\n")
					out.WriteString(mermaidFence + "\n")
				} else {
					rendered := strings.TrimRight(res.Output, "\n")
					out.WriteString(rendered)
					out.WriteString("\n")
				}
				continue
			}
			mermaidLines = append(mermaidLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan markdown: %w", err)
	}

	// If a mermaid block was unclosed at EOF, render or emit it
	if inMermaidBlock && len(mermaidLines) > 0 {
		mermaidSource := strings.Join(mermaidLines, "\n")
		res, err := p.Render(ctx, mermaidSource)
		if err == nil {
			out.WriteString(strings.TrimRight(res.Output, "\n"))
			out.WriteString("\n")
		} else {
			out.WriteString(mermaidFence + "mermaid\n")
			out.WriteString(mermaidSource + "\n")
		}
	}

	return out.String(), nil
}

// PrintMarkdown renders all Mermaid blocks in the given markdown string to the configured Writer.
func (p *Printer) PrintMarkdown(ctx context.Context, markdown string) error {
	rendered, err := p.RenderMarkdown(ctx, markdown)
	if err != nil {
		return err
	}
	_, err = io.WriteString(p.config.Writer, rendered)
	return err
}

// PrintMarkdownFile reads a markdown file and prints it with rendered Mermaid diagrams.
func (p *Printer) PrintMarkdownFile(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read markdown file %q: %w", filePath, err)
	}
	return p.PrintMarkdown(ctx, string(data))
}

// RenderMarkdown parses and renders Mermaid diagrams in markdown using default settings.
func RenderMarkdown(ctx context.Context, markdown string, opts ...Option) (string, error) {
	return New(opts...).RenderMarkdown(ctx, markdown)
}

// PrintMarkdown renders and prints a markdown string with Mermaid diagrams using default settings.
func PrintMarkdown(markdown string, opts ...Option) error {
	return New(opts...).PrintMarkdown(context.Background(), markdown)
}

// PrintMarkdownContext renders and prints markdown with context.
func PrintMarkdownContext(ctx context.Context, markdown string, opts ...Option) error {
	return New(opts...).PrintMarkdown(ctx, markdown)
}

// PrintMarkdownFile reads and prints a markdown file with rendered Mermaid diagrams.
func PrintMarkdownFile(filePath string, opts ...Option) error {
	return New(opts...).PrintMarkdownFile(context.Background(), filePath)
}

// PrintMarkdownFileContext reads and prints a markdown file with context.
func PrintMarkdownFileContext(ctx context.Context, filePath string, opts ...Option) error {
	return New(opts...).PrintMarkdownFile(ctx, filePath)
}

// IsMarkdownFile checks if a file path has a markdown extension (.md, .markdown, .mdown).
func IsMarkdownFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".md" || ext == ".markdown" || ext == ".mdown"
}

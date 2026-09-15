package mermaid

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"runbook.md", true},
		{"doc.markdown", true},
		{"notes.mdown", true},
		{"UPPERCASE.MD", true},
		{"diagram.mmd", false},
		{"diagram.mermaid", false},
		{"main.go", false},
		{"README", false},
	}

	for _, tc := range tests {
		got := IsMarkdownFile(tc.path)
		if got != tc.expected {
			t.Errorf("IsMarkdownFile(%q) = %v, expected %v", tc.path, got, tc.expected)
		}
	}
}

func TestRenderMarkdown_NoMermaid(t *testing.T) {
	p := New(WithMode(ModeASCII))
	input := "# Runbook Header\n\nSome incident notes.\n\n```go\nfmt.Println(\"hello\")\n```\n"

	out, err := p.RenderMarkdown(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != input {
		t.Errorf("expected markdown without mermaid to remain identical.\nGot:\n%s\nWant:\n%s", out, input)
	}
}

func TestRenderMarkdown_SingleMermaidBlock(t *testing.T) {
	p := New(WithMode(ModeASCII))
	input := "# Service Architecture\n\nOverview diagram:\n\n```mermaid\ngraph TD\n  ServiceA --> ServiceB\n```\n\nFollow up notes.\n"

	out, err := p.RenderMarkdown(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "# Service Architecture") {
		t.Error("expected markdown header to be preserved")
	}
	if !strings.Contains(out, "Follow up notes.") {
		t.Error("expected markdown notes to be preserved")
	}
	if !strings.Contains(out, "ServiceA") || !strings.Contains(out, "ServiceB") {
		t.Errorf("expected rendered diagram to contain ServiceA and ServiceB, got:\n%s", out)
	}
	if strings.Contains(out, "```mermaid") {
		t.Error("expected ```mermaid fence to be replaced by rendered diagram")
	}
}

func TestRenderMarkdown_MultipleMermaidBlocks(t *testing.T) {
	p := New(WithMode(ModeASCII))
	input := `## Section 1
` + "```mermaid" + `
graph TD
  Node1 --> Node2
` + "```" + `

## Section 2
` + "~~~mermaid" + `
graph TD
  Node3 --> Node4
` + "~~~" + `

Done.
`

	out, err := p.RenderMarkdown(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Node1") || !strings.Contains(out, "Node2") {
		t.Error("expected section 1 diagram to be rendered")
	}
	if !strings.Contains(out, "Node3") || !strings.Contains(out, "Node4") {
		t.Error("expected section 2 diagram to be rendered")
	}
	if !strings.Contains(out, "## Section 1") || !strings.Contains(out, "## Section 2") {
		t.Error("expected sections to be preserved")
	}
}

func TestPrintMarkdownFile(t *testing.T) {
	buf := new(bytes.Buffer)
	p := New(
		WithWriter(buf),
		WithMode(ModeASCII),
	)

	if err := p.PrintMarkdownFile(context.Background(), "testdata/runbook.md"); err != nil {
		t.Fatalf("PrintMarkdownFile failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "SRE Incident Response Runbook") {
		t.Errorf("expected output to contain runbook title, got:\n%s", output)
	}
	if !strings.Contains(output, "PostgreSQL Primary") {
		t.Errorf("expected output to contain rendered diagram nodes, got:\n%s", output)
	}
}

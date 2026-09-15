package mermaid

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTestDataDiagrams(t *testing.T) {
	files, err := filepath.Glob("testdata/*.mmd")
	if err != nil {
		t.Fatalf("failed to glob testdata: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no testdata .mmd files found")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read %s: %v", file, err)
			}

			// 1. Test ASCII text rendering
			buf := new(bytes.Buffer)
			printer := New(
				WithWriter(buf),
				WithMode(ModeASCII),
			)

			res, err := printer.Render(context.Background(), string(content))
			if err != nil {
				t.Fatalf("render ASCII failed for %s: %v", file, err)
			}
			if len(res.Output) == 0 {
				t.Errorf("empty output for %s", file)
			}

			// 2. Test PrintFile
			buf.Reset()
			err = printer.PrintFile(file)
			if err != nil {
				t.Fatalf("PrintFile failed for %s: %v", file, err)
			}
			if buf.Len() == 0 {
				t.Errorf("empty buffer after PrintFile for %s", file)
			}
		})
	}
}

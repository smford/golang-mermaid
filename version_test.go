package mermaid_test

import (
	"strings"
	"testing"

	mermaid "github.com/smford/golang-mermaid"
)

func TestVersion(t *testing.T) {
	if mermaid.Version == "" {
		t.Fatal("expected Version to be non-empty")
	}
	parts := strings.Split(mermaid.Version, ".")
	if len(parts) != 3 {
		t.Fatalf("expected SemVer format (MAJOR.MINOR.PATCH), got %q", mermaid.Version)
	}
}

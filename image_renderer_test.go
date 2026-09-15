package mermaid

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKrokiRenderer_Success(t *testing.T) {
	expectedBytes := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A} // PNG magic bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/mermaid/png" {
			t.Errorf("expected /mermaid/png, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(expectedBytes)
	}))
	defer server.Close()

	renderer := NewKrokiRenderer(server.URL, 2*time.Second)
	data, err := renderer.RenderImage(context.Background(), "graph TD; A-->B;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(expectedBytes) {
		t.Fatalf("expected %v, got %v", expectedBytes, data)
	}
}

func TestKrokiRenderer_EmptySource(t *testing.T) {
	renderer := NewKrokiRenderer("https://kroki.io", 2*time.Second)
	_, err := renderer.RenderImage(context.Background(), "   ")
	if !errors.Is(err, ErrEmptySource) {
		t.Fatalf("expected ErrEmptySource, got %v", err)
	}
}

func TestKrokiRenderer_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Syntax error in diagram"))
	}))
	defer server.Close()

	renderer := NewKrokiRenderer(server.URL, 2*time.Second)
	_, err := renderer.RenderImage(context.Background(), "invalid diagram syntax")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMermaidInkRenderer_Success(t *testing.T) {
	expectedBytes := []byte("FAKE_INK_IMAGE")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(expectedBytes)
	}))
	defer server.Close()

	renderer := NewMermaidInkRenderer(2 * time.Second)
	renderer.BaseURL = server.URL

	data, err := renderer.RenderImage(context.Background(), "graph TD; A-->B;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(expectedBytes) {
		t.Fatalf("expected %v, got %v", expectedBytes, data)
	}
}

type mockImageRenderer struct {
	data []byte
	err  error
}

func (m *mockImageRenderer) RenderImage(ctx context.Context, mermaidSource string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func TestResilientImageRenderer_FallbackChain(t *testing.T) {
	failRenderer1 := &mockImageRenderer{err: errors.New("network down")}
	failRenderer2 := &mockImageRenderer{err: errors.New("timeout")}
	successRenderer := &mockImageRenderer{data: []byte("SUCCESS_IMAGE")}

	chain := &ResilientImageRenderer{
		Renderers: []ImageRenderer{failRenderer1, failRenderer2, successRenderer},
	}

	data, err := chain.RenderImage(context.Background(), "graph TD; A-->B;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "SUCCESS_IMAGE" {
		t.Fatalf("expected SUCCESS_IMAGE, got %s", string(data))
	}
}

func TestResilientImageRenderer_AllFail(t *testing.T) {
	failRenderer1 := &mockImageRenderer{err: errors.New("error 1")}
	failRenderer2 := &mockImageRenderer{err: errors.New("error 2")}

	chain := &ResilientImageRenderer{
		Renderers: []ImageRenderer{failRenderer1, failRenderer2},
	}

	_, err := chain.RenderImage(context.Background(), "graph TD; A-->B;")
	if err == nil {
		t.Fatal("expected error when all fail, got nil")
	}
	if !errors.Is(err, ErrAllRenderers) {
		t.Fatalf("expected ErrAllRenderers, got: %v", err)
	}
}

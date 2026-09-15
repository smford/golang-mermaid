package mermaid_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	mermaid "github.com/smford/golang-mermaid"
)

func TestStatsRecorder_RecordAndExport(t *testing.T) {
	stats := mermaid.NewStatsRecorder()

	// Initial state
	if stats.TotalRenders != 0 {
		t.Fatalf("expected 0 total renders, got %d", stats.TotalRenders)
	}
	if stats.AverageDuration() != 0 {
		t.Fatalf("expected 0 average duration, got %v", stats.AverageDuration())
	}

	ctx := context.Background()

	// Record an image render
	stats.RecordRender(ctx, mermaid.TelemetryEvent{
		Duration:         50 * time.Millisecond,
		Mode:             mermaid.ModeImage,
		Protocol:         mermaid.ProtocolKitty,
		FallbackOccurred: false,
		CacheHit:         false,
		SourceLength:     100,
	})

	// Record a fallback text render
	stats.RecordRender(ctx, mermaid.TelemetryEvent{
		Duration:         10 * time.Millisecond,
		Mode:             mermaid.ModeUnicode,
		FallbackOccurred: true,
		FallbackReason:   "network timeout",
		CacheHit:         false,
		SourceLength:     80,
	})

	// Record a cache hit
	stats.RecordRender(ctx, mermaid.TelemetryEvent{
		Duration:         1 * time.Millisecond,
		Mode:             mermaid.ModeImage,
		Protocol:         mermaid.ProtocolKitty,
		FallbackOccurred: false,
		CacheHit:         true,
		SourceLength:     100,
	})

	if stats.TotalRenders != 3 {
		t.Errorf("expected 3 total renders, got %d", stats.TotalRenders)
	}
	if stats.ImageRenders != 2 {
		t.Errorf("expected 2 image renders, got %d", stats.ImageRenders)
	}
	if stats.TextRenders != 1 {
		t.Errorf("expected 1 text render, got %d", stats.TextRenders)
	}
	if stats.Fallbacks != 1 {
		t.Errorf("expected 1 fallback, got %d", stats.Fallbacks)
	}
	if stats.CacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats.CacheHits)
	}
	if stats.Reasons["network timeout"] != 1 {
		t.Errorf("expected 1 network timeout reason, got %d", stats.Reasons["network timeout"])
	}

	avg := stats.AverageDuration()
	expectedAvg := (50 + 10 + 1) * time.Millisecond / 3
	if avg != expectedAvg {
		t.Errorf("expected average duration %v, got %v", expectedAvg, avg)
	}

	prom := stats.ExportPrometheus()
	requiredMetrics := []string{
		"mermaid_renders_total 3",
		`mermaid_renders_by_mode_total{mode="image"} 2`,
		`mermaid_renders_by_mode_total{mode="text"} 1`,
		"mermaid_fallbacks_total 1",
		"mermaid_cache_hits_total 1",
		"mermaid_render_duration_seconds_total",
	}

	for _, m := range requiredMetrics {
		if !strings.Contains(prom, m) {
			t.Errorf("expected Prometheus output to contain %q, but got:\n%s", m, prom)
		}
	}
}

func TestStatsRecorder_Concurrency(t *testing.T) {
	stats := mermaid.NewStatsRecorder()
	ctx := context.Background()

	var wg sync.WaitGroup
	workers := 20
	iterations := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				stats.RecordRender(ctx, mermaid.TelemetryEvent{
					Duration:         time.Duration(j) * time.Millisecond,
					Mode:             mermaid.ModeUnicode,
					FallbackOccurred: j%2 == 0,
					FallbackReason:   "test-fallback",
					CacheHit:         j%3 == 0,
					SourceLength:     50,
				})
			}
		}(i)
	}

	wg.Wait()

	expectedTotal := uint64(workers * iterations)
	if stats.TotalRenders != expectedTotal {
		t.Fatalf("expected %d total renders, got %d", expectedTotal, stats.TotalRenders)
	}
}

func TestPrinter_WithTelemetry(t *testing.T) {
	var recordedEvents []mermaid.TelemetryEvent
	var mu sync.Mutex

	customRecorder := mermaid.TelemetryFunc(func(ctx context.Context, event mermaid.TelemetryEvent) {
		mu.Lock()
		defer mu.Unlock()
		recordedEvents = append(recordedEvents, event)
	})

	diagram := "graph TD\n  A --> B"
	printer := mermaid.New(
		mermaid.WithMode(mermaid.ModeASCII),
		mermaid.WithTelemetry(customRecorder),
	)

	_, err := printer.Render(context.Background(), diagram)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(recordedEvents) != 1 {
		t.Fatalf("expected 1 recorded telemetry event, got %d", len(recordedEvents))
	}

	evt := recordedEvents[0]
	if evt.Mode != mermaid.ModeASCII {
		t.Errorf("expected ModeASCII, got %v", evt.Mode)
	}
	if evt.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", evt.Duration)
	}
	if evt.SourceLength != len(diagram) {
		t.Errorf("expected source length %d, got %d", len(diagram), evt.SourceLength)
	}
}

func TestPrinter_WithTelemetryFunc(t *testing.T) {
	var called bool
	printer := mermaid.New(
		mermaid.WithMode(mermaid.ModeASCII),
		mermaid.WithTelemetryFunc(func(ctx context.Context, event mermaid.TelemetryEvent) {
			called = true
		}),
	)

	_, err := printer.Render(context.Background(), "graph TD\n  A --> B")
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !called {
		t.Errorf("expected WithTelemetryFunc callback to be called")
	}
}

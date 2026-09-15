package mermaid

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TelemetryEvent captures structured performance and operational metrics for a diagram render.
type TelemetryEvent struct {
	Duration         time.Duration
	Mode             RenderMode
	Protocol         GraphicsProtocol
	FallbackOccurred bool
	FallbackReason   string
	CacheHit         bool
	SourceLength     int
}

// TelemetryRecorder is the interface implemented by monitoring systems (OpenTelemetry, Prometheus, Datadog).
type TelemetryRecorder interface {
	RecordRender(ctx context.Context, event TelemetryEvent)
}

// TelemetryFunc is an adapter allowing a plain function to act as a TelemetryRecorder.
type TelemetryFunc func(ctx context.Context, event TelemetryEvent)

// RecordRender implements TelemetryRecorder.
func (f TelemetryFunc) RecordRender(ctx context.Context, event TelemetryEvent) {
	f(ctx, event)
}

// StatsRecorder is a thread-safe SRE operational metrics accumulator.
type StatsRecorder struct {
	mu           sync.RWMutex
	TotalRenders uint64
	ImageRenders uint64
	TextRenders  uint64
	Fallbacks    uint64
	CacheHits    uint64
	TotalNanos   uint64
	Reasons      map[string]uint64
}

// NewStatsRecorder creates a new thread-safe metrics recorder.
func NewStatsRecorder() *StatsRecorder {
	return &StatsRecorder{
		Reasons: make(map[string]uint64),
	}
}

// RecordRender records an individual render event into the metrics accumulator.
func (s *StatsRecorder) RecordRender(ctx context.Context, event TelemetryEvent) {
	atomic.AddUint64(&s.TotalRenders, 1)
	atomic.AddUint64(&s.TotalNanos, uint64(event.Duration.Nanoseconds()))

	if event.Mode == ModeImage {
		atomic.AddUint64(&s.ImageRenders, 1)
	} else {
		atomic.AddUint64(&s.TextRenders, 1)
	}

	if event.FallbackOccurred {
		atomic.AddUint64(&s.Fallbacks, 1)
	}
	if event.CacheHit {
		atomic.AddUint64(&s.CacheHits, 1)
	}

	if event.FallbackReason != "" {
		s.mu.Lock()
		s.Reasons[event.FallbackReason]++
		s.mu.Unlock()
	}
}

// AverageDuration returns the average latency across all recorded renders.
func (s *StatsRecorder) AverageDuration() time.Duration {
	total := atomic.LoadUint64(&s.TotalRenders)
	if total == 0 {
		return 0
	}
	nanos := atomic.LoadUint64(&s.TotalNanos)
	return time.Duration(nanos / total)
}

// ExportPrometheus exports the accumulated metrics in Prometheus exposition text format.
func (s *StatsRecorder) ExportPrometheus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("# HELP mermaid_renders_total Total number of diagram renders.\n")
	sb.WriteString("# TYPE mermaid_renders_total counter\n")
	sb.WriteString(fmt.Sprintf("mermaid_renders_total %d\n", atomic.LoadUint64(&s.TotalRenders)))

	sb.WriteString("# HELP mermaid_renders_by_mode_total Total number of renders by mode.\n")
	sb.WriteString("# TYPE mermaid_renders_by_mode_total counter\n")
	sb.WriteString(fmt.Sprintf("mermaid_renders_by_mode_total{mode=\"image\"} %d\n", atomic.LoadUint64(&s.ImageRenders)))
	sb.WriteString(fmt.Sprintf("mermaid_renders_by_mode_total{mode=\"text\"} %d\n", atomic.LoadUint64(&s.TextRenders)))

	sb.WriteString("# HELP mermaid_fallbacks_total Total number of image fallbacks triggered.\n")
	sb.WriteString("# TYPE mermaid_fallbacks_total counter\n")
	sb.WriteString(fmt.Sprintf("mermaid_fallbacks_total %d\n", atomic.LoadUint64(&s.Fallbacks)))

	sb.WriteString("# HELP mermaid_cache_hits_total Total number of diagram cache hits.\n")
	sb.WriteString("# TYPE mermaid_cache_hits_total counter\n")
	sb.WriteString(fmt.Sprintf("mermaid_cache_hits_total %d\n", atomic.LoadUint64(&s.CacheHits)))

	sb.WriteString("# HELP mermaid_render_duration_seconds_total Total duration of diagram renders in seconds.\n")
	sb.WriteString("# TYPE mermaid_render_duration_seconds_total counter\n")
	secs := float64(atomic.LoadUint64(&s.TotalNanos)) / 1e9
	sb.WriteString(fmt.Sprintf("mermaid_render_duration_seconds_total %.6f\n", secs))

	return sb.String()
}

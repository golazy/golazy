//go:build lazydev

package lazytracing

import (
	"context"
	"runtime"
	"sync"
)

type allocationSamplingKey struct{}

type allocationSample struct {
	mu    sync.Mutex
	start runtime.MemStats
	end   runtime.MemStats
	ended bool
}

// AllocationSummary contains cumulative process allocation deltas observed
// while a lazydev span was active.
type AllocationSummary struct {
	TotalAllocBytesDelta uint64
	MallocsDelta         uint64
	FreesDelta           uint64
}

var allocationSamples sync.Map

// WithAllocationSampling enables lazydev allocation sampling for spans started
// from ctx.
func WithAllocationSampling(ctx context.Context) context.Context {
	return context.WithValue(ctx, allocationSamplingKey{}, true)
}

func allocationSamplingEnabled(ctx context.Context) bool {
	enabled, _ := ctx.Value(allocationSamplingKey{}).(bool)
	return enabled
}

func startSpanAllocationSample(ctx context.Context, span *Span) {
	if span == nil || !allocationSamplingEnabled(ctx) {
		return
	}
	var start runtime.MemStats
	runtime.ReadMemStats(&start)
	allocationSamples.Store(span, &allocationSample{start: start})
}

func finishSpanAllocationSample(span *Span) {
	if span == nil {
		return
	}
	value, ok := allocationSamples.Load(span)
	if !ok {
		return
	}
	sample, ok := value.(*allocationSample)
	if !ok || sample == nil {
		return
	}
	sample.mu.Lock()
	defer sample.mu.Unlock()
	if sample.ended {
		return
	}
	runtime.ReadMemStats(&sample.end)
	sample.ended = true
}

// SpanAllocationSummary returns the allocation delta sampled for span.
func SpanAllocationSummary(span *Span) (AllocationSummary, bool) {
	if span == nil {
		return AllocationSummary{}, false
	}
	value, ok := allocationSamples.Load(span)
	if !ok {
		return AllocationSummary{}, false
	}
	sample, ok := value.(*allocationSample)
	if !ok || sample == nil {
		return AllocationSummary{}, false
	}
	sample.mu.Lock()
	defer sample.mu.Unlock()
	if !sample.ended {
		return AllocationSummary{}, false
	}
	return AllocationSummary{
		TotalAllocBytesDelta: uint64Delta(sample.end.TotalAlloc, sample.start.TotalAlloc),
		MallocsDelta:         uint64Delta(sample.end.Mallocs, sample.start.Mallocs),
		FreesDelta:           uint64Delta(sample.end.Frees, sample.start.Frees),
	}, true
}

// ClearAllocationSamples removes lazydev allocation samples for span and its
// descendants after request capture has been written.
func ClearAllocationSamples(span *Span) {
	if span == nil {
		return
	}
	for _, child := range span.Children() {
		ClearAllocationSamples(child)
	}
	allocationSamples.Delete(span)
}

func uint64Delta(end, start uint64) uint64 {
	if end < start {
		return 0
	}
	return end - start
}

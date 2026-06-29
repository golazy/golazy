package lazyview

import "testing"

func TestPooledRenderBufferClearsState(t *testing.T) {
	first := acquireRenderBuffer()
	first.WriteString("stale")
	releaseRenderBuffer(first)

	second := acquireRenderBuffer()
	defer releaseRenderBuffer(second)
	if got := second.Len(); got != 0 {
		t.Fatalf("pooled render buffer length = %d, want 0", got)
	}
}

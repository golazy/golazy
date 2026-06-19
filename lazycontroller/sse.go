package lazycontroller

import (
	"fmt"

	"golazy.dev/lazysse"
)

// SSEStream starts a Server-Sent Events response for the current controller
// request.
func (b *Base) SSEStream(opts ...lazysse.Option) (*lazysse.Stream, error) {
	if b == nil || b.writer == nil || b.request == nil {
		return nil, fmt.Errorf("lazycontroller: controller base is not initialized")
	}
	if b.status != 0 {
		opts = append([]lazysse.Option{lazysse.Status(b.status)}, opts...)
	}
	return lazysse.Start(b.writer, b.request, opts...)
}

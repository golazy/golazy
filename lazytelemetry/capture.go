package lazytelemetry

import "time"

type requestCaptureResult struct {
	RequestID string
	Method    string
	Path      string
	Status    int
	Bytes     int
	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration
	Panic     any
}

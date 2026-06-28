package lazyjobs

import (
	"context"
	"encoding/json"
	"time"
)

type Backend interface {
	Insert(context.Context, InsertParams) (Record, error)
	Claim(context.Context, ClaimParams) (Record, bool, error)
	Complete(context.Context, int64) error
	Retry(context.Context, RetryParams) error
	Discard(context.Context, DiscardParams) error
	List(context.Context, ListOptions) ([]Record, error)
	Stats(context.Context) (Stats, error)
}

type InsertParams struct {
	Kind        string
	Queue       string
	Payload     json.RawMessage
	MaxAttempts int
	RunAt       time.Time
}

type ClaimParams struct {
	Queues []string
	Now    time.Time
}

type RetryParams struct {
	ID        int64
	RunAt     time.Time
	LastError string
}

type DiscardParams struct {
	ID        int64
	LastError string
}

type ListOptions struct {
	Limit int
}

type closeBackend interface {
	Close() error
}

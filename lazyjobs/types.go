package lazyjobs

import (
	"context"
	"encoding/json"
	"time"
)

const DefaultQueue = "default"

type State string

const (
	StatePending   State = "pending"
	StateRunning   State = "running"
	StateRetrying  State = "retrying"
	StateSucceeded State = "succeeded"
	StateDiscarded State = "discarded"
)

type Job interface {
	Kind() string
	Work(context.Context) error
}

type QueueNamer interface {
	JobQueue() string
}

type RetryPolicy interface {
	JobMaxAttempts() int
	JobRetryDelay(attempt int, err error) time.Duration
}

type Record struct {
	ID          int64           `json:"id"`
	Kind        string          `json:"kind"`
	Queue       string          `json:"queue"`
	Payload     json.RawMessage `json:"-"`
	State       State           `json:"state"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"max_attempts"`
	RunAt       time.Time       `json:"run_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	LastError   string          `json:"last_error,omitempty"`
}

type Definition struct {
	Kind        string `json:"kind"`
	Type        string `json:"type"`
	Queue       string `json:"queue"`
	MaxAttempts int    `json:"max_attempts"`
}

type Stats struct {
	Total   int            `json:"total"`
	ByState map[State]int  `json:"by_state"`
	ByKind  map[string]int `json:"by_kind"`
	ByQueue map[string]int `json:"by_queue"`
}

type Snapshot struct {
	Running     bool         `json:"running"`
	Definitions []Definition `json:"definitions"`
	Stats       Stats        `json:"stats"`
	Recent      []Record     `json:"recent"`
}

func normalizeQueue(queue string) string {
	if queue == "" {
		return DefaultQueue
	}
	return queue
}

func normalizeAttempts(attempts int) int {
	if attempts <= 0 {
		return 25
	}
	return attempts
}

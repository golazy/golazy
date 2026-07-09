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
	ScheduleKey string
	Payload     json.RawMessage
	MaxAttempts int
	RunAt       time.Time
}

type ClaimParams struct {
	Queues      []string
	QueueLimits map[string]int
	Now         time.Time
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

type SchedulerBackend interface {
	RegisterSchedule(context.Context, ScheduleParams) (ScheduleRecord, error)
	ClaimSchedule(context.Context, ClaimScheduleParams) (ScheduleRecord, bool, error)
	AdvanceSchedule(context.Context, AdvanceScheduleParams) error
	HasActiveScheduledJob(context.Context, ActiveScheduledJobParams) (bool, error)
	ListSchedules(context.Context) ([]ScheduleRecord, error)
}

type ScheduleParams struct {
	Key       string
	Kind      string
	Queue     string
	Payload   json.RawMessage
	Interval  time.Duration
	NextRunAt time.Time
}

type ClaimScheduleParams struct {
	Now time.Time
}

type AdvanceScheduleParams struct {
	Key       string
	NextRunAt time.Time
}

type ActiveScheduledJobParams struct {
	ScheduleKey string
}

type ScheduleRecord struct {
	Key       string
	Kind      string
	Queue     string
	Payload   json.RawMessage
	Interval  time.Duration
	NextRunAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type closeBackend interface {
	Close() error
}

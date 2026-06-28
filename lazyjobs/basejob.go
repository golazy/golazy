package lazyjobs

import "time"

type BaseJob struct{}

func (BaseJob) JobQueue() string {
	return DefaultQueue
}

func (BaseJob) JobMaxAttempts() int {
	return 25
}

func (BaseJob) JobRetryDelay(attempt int, _ error) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Duration(1<<min(attempt-1, 6)) * time.Second
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}

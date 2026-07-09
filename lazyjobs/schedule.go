package lazyjobs

import "time"

type Schedule struct {
	Key        string
	Interval   time.Duration
	Job        Job
	Queue      string
	FirstRunAt time.Time
}

func Every(key string, interval time.Duration, job Job, options ...ScheduleOption) Schedule {
	scheduleOptions := scheduleOptions{}
	for _, option := range options {
		if option != nil {
			option.applyScheduleOption(&scheduleOptions)
		}
	}
	return Schedule{
		Key:        key,
		Interval:   interval,
		Job:        job,
		Queue:      scheduleOptions.queue,
		FirstRunAt: scheduleOptions.firstRunAt,
	}
}

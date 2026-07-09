package lazyjobs

import "time"

type EnqueueOption interface {
	applyEnqueueOption(*enqueueOptions)
}

type ScheduleOption interface {
	applyScheduleOption(*scheduleOptions)
}

type enqueueOptions struct {
	queue       string
	runAt       time.Time
	scheduleKey string
}

type scheduleOptions struct {
	queue      string
	firstRunAt time.Time
}

type queueOption string

func Queue(name string) queueOption {
	return queueOption(name)
}

func (option queueOption) applyEnqueueOption(options *enqueueOptions) {
	options.queue = normalizeQueue(string(option))
}

func (option queueOption) applyScheduleOption(options *scheduleOptions) {
	options.queue = normalizeQueue(string(option))
}

type runAtOption time.Time

func RunAt(runAt time.Time) runAtOption {
	return runAtOption(runAt)
}

func (option runAtOption) applyEnqueueOption(options *enqueueOptions) {
	options.runAt = time.Time(option)
}

func (option runAtOption) applyScheduleOption(options *scheduleOptions) {
	options.firstRunAt = time.Time(option)
}

type runInOption time.Duration

func RunIn(delay time.Duration) runInOption {
	return runInOption(delay)
}

func (option runInOption) applyEnqueueOption(options *enqueueOptions) {
	delay := time.Duration(option)
	if delay < 0 {
		delay = 0
	}
	options.runAt = time.Now().UTC().Add(delay)
}

func (option runInOption) applyScheduleOption(options *scheduleOptions) {
	delay := time.Duration(option)
	if delay < 0 {
		delay = 0
	}
	options.firstRunAt = time.Now().UTC().Add(delay)
}

type scheduleKeyOption string

func (option scheduleKeyOption) applyEnqueueOption(options *enqueueOptions) {
	options.scheduleKey = string(option)
}

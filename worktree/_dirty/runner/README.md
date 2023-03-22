# runner

Package runner run a restart a program on signals

## Variables

```golang
var (
    // ErrRunning is the return error in the Start method
    ErrRunning = errors.New("Program is already running")
    // ErrCantKill is returned by Restart and Stop in case the process can't be killed
    ErrCantKill = errors.New("Process is still alive after sending the kill signal")
    // ErrNotRunning is retuned by the Stop and Signal command when the program is not running
    ErrNotRunning = errors.New("Process is not running")
    // ErrRunnerClosed is returned by any method when the runner is closed
    ErrRunnerClosed = errors.New("Runner is closed")
)
```

DefaultRunnerOptions are used when no RunnerOptions are passed

```golang
var DefaultRunnerOptions = &Options{
    KillWaitPeriod: time.Second,
    ReadyString:    []string{"Listening", "Started", "Ready"},
}
```

## Types

### type [EventReady](/runner.go#L45)

`type EventReady struct { ... }`

EventReady is fired whenever the command outputs the string Listening

### type [EventRestart](/runner.go#L50)

`type EventRestart struct { ... }`

EventRestart is fired whenever Restart is called

### type [EventSignal](/runner.go#L39)

`type EventSignal struct { ... }`

EventSignal is fired whenever

### type [EventStart](/runner.go#L26)

`type EventStart struct { ... }`

EventStart is fired whenever Start is called

### type [EventStarted](/runner.go#L64)

`type EventStarted struct { ... }`

EventStarted is fired whenever the subprocess is started

### type [EventStop](/runner.go#L33)

`type EventStop struct { ... }`

EventStop is fired whenever Stop is called

### type [EventStopped](/runner.go#L57)

`type EventStopped struct { ... }`

EventStopped is fired whenever the process exits

### type [Options](/runner.go#L14)

`type Options struct { ... }`

Options holds the runner options

### type [Runner](/runner.go#L69)

`type Runner struct { ... }`

Runner is an command runner that produces events on start/stop and restart

#### func [New](/runner.go#L145)

`func New(cmd *exec.Cmd, options *Options) *Runner`

New creates a new runner for the given command
if options is nil, New will use DefaultRunnerOptions

#### func (*Runner) [Close](/runner.go#L87)

`func (r *Runner) Close() error`

Close stop all the internal goroutines
After Close is called the runner can't be used anymore

#### func (*Runner) [Restart](/runner.go#L108)

`func (r *Runner) Restart() error`

Restart restart the process by calling Stop and then Restart. If the process is not runing it will be the same as calling Start

#### func (*Runner) [Signal](/runner.go#L131)

`func (r *Runner) Signal(s syscall.Signal) error`

Signal sends a signal to the process.
If the process is not running it returns ErrNotRunning

#### func (*Runner) [Start](/runner.go#L98)

`func (r *Runner) Start() error`

Start starts the command
If the command is already running it returns ErrRunning

#### func (*Runner) [Stop](/runner.go#L120)

`func (r *Runner) Stop() error`

Stop stops the process.
It will send an interrupt signal to the process.
If after KillWaitPeriod the process is still alive, it will send a kill signal

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)

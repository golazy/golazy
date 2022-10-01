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
    KillWaitPeriod:  time.Second,
    ReadyString:     []string{"Listening", "Started"},
    CopyOutputToStd: true,
}
```

## Types

### type [EventReady](/runner.go#L37)

`type EventReady struct { ... }`

EventReady is fired whenever the command outputs the string Listening

### type [EventRestart](/runner.go#L42)

`type EventRestart struct{ ... }`

EventRestart is fired whenever Restart is called

### type [EventSignal](/runner.go#L34)

`type EventSignal struct{ ... }`

EventSignal is fired whenever

### type [EventStart](/runner.go#L28)

`type EventStart struct{ ... }`

EventStart is fired whenever Start is called

### type [EventStop](/runner.go#L31)

`type EventStop struct{ ... }`

EventStop is fired whenever Stop is called

### type [EventStopped](/runner.go#L45)

`type EventStopped struct { ... }`

EventStopped is fired whenever the process exits

### type [Options](/runner.go#L14)

`type Options struct { ... }`

Options holds the runner options

### type [Runner](/runner.go#L52)

`type Runner struct { ... }`

Runner is an command runner that produces events on start/stop and restart

#### func [New](/runner.go#L128)

`func New(cmd *exec.Cmd, options *Options) *Runner`

New creates a new runner for the given command
if options is nil, New will use DefaultRunnerOptions

#### func (*Runner) [Close](/runner.go#L70)

`func (r *Runner) Close() error`

Close stop all the internal goroutines
After Close is called the runner can't be used anymore

#### func (*Runner) [Restart](/runner.go#L91)

`func (r *Runner) Restart() error`

Restart restart the process by calling Stop and then Restart. If the process is not runing it will be the same as calling Start

#### func (*Runner) [Signal](/runner.go#L114)

`func (r *Runner) Signal(s os.Signal) error`

Signal sends a signal to the process.
If the process is not running it returns ErrNotRunning

#### func (*Runner) [Start](/runner.go#L81)

`func (r *Runner) Start() error`

Start starts the command
If the command is already running it returns ErrRunning

#### func (*Runner) [Stop](/runner.go#L103)

`func (r *Runner) Stop() error`

Stop stops the process.
It will send an interrupt signal to the process.
If after KillWaitPeriod the process is still alive, it will send a kill signal

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)

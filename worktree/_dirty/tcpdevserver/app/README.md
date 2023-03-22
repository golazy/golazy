# app

## Types

### type [BuildError](/build.go#L22)

`type BuildError buildResult`

#### func (BuildError) [Error](/build.go#L24)

`func (b BuildError) Error() string`

#### func (BuildError) [Unwrap](/build.go#L28)

`func (b BuildError) Unwrap() error`

### type [Cmd](/cmd.go#L9)

`type Cmd struct { ... }`

#### func (*Cmd) [Start](/cmd.go#L17)

`func (c *Cmd) Start() error`

#### func (*Cmd) [Stop](/cmd.go#L33)

`func (c *Cmd) Stop() error`

### type [Event](/event.go#L8)

`type Event interface { ... }`

### type [EventAppBuildFailure](/event.go#L87)

`type EventAppBuildFailure struct { ... }`

### type [EventAppBuildFinish](/event.go#L81)

`type EventAppBuildFinish struct { ... }`

EventAppBuildFinish fires when the build finished regarding the build status

### type [EventAppBuildStart](/event.go#L76)

`type EventAppBuildStart struct { ... }`

### type [EventAppBuildSuccess](/event.go#L93)

`type EventAppBuildSuccess struct { ... }`

### type [EventAppStart](/event.go#L38)

`type EventAppStart struct { ... }`

### type [EventAppStartFail](/event.go#L44)

`type EventAppStartFail struct { ... }`

### type [EventAppStderr](/event.go#L54)

`type EventAppStderr struct { ... }`

### type [EventAppStdout](/event.go#L49)

`type EventAppStdout struct { ... }`

### type [EventAppStop](/event.go#L64)

`type EventAppStop struct { ... }`

### type [EventAppStopping](/event.go#L59)

`type EventAppStopping struct { ... }`

### type [EventAppUnexpectedExit](/event.go#L72)

`type EventAppUnexpectedExit struct { ... }`

Uninplemented

### type [EventBase](/event.go#L21)

`type EventBase struct { ... }`

#### func (EventBase) [CreatedAt](/event.go#L30)

`func (e EventBase) CreatedAt() time.Time`

#### func (EventBase) [Event](/event.go#L34)

`func (e EventBase) Event() string`

#### func (EventBase) [Name](/event.go#L26)

`func (e EventBase) Name() string`

### type [GoApp](/app.go#L66)

`type GoApp struct { ... }`

#### func [New](/app.go#L16)

`func New(opt GoAppOptions) *GoApp`

#### func (*GoApp) [Clean](/app.go#L147)

`func (app *GoApp) Clean()`

#### func (*GoApp) [Start](/app.go#L78)

`func (app *GoApp) Start()`

#### func (*GoApp) [Stop](/app.go#L142)

`func (app *GoApp) Stop()`

### type [GoAppOptions](/app.go#L10)

`type GoAppOptions struct { ... }`

## Sub Packages

* [test](./test)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)

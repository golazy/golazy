# tcpdevserver

## Types

### type [Runner](/runner.go#L13)

`type Runner struct { ... }`

```golang
var count int
r := &Runner{
    Command: func() *exec.Cmd {
        count += 1
        cmd := exec.Command("bash", "-c", fmt.Sprintf(" echo %d ; sleep %d", count, count))
        cmd.Stdout = os.Stdout
        return cmd
    },
}

r.Start() // 1
time.Sleep(time.Millisecond * 200)

r.Restart() // 2
time.Sleep(time.Millisecond * 200)

r.Restart() // 3
time.Sleep(time.Millisecond * 200)

r.Restart() // 4
time.Sleep(time.Millisecond * 200)

r.Stop() // 4

//r.Stop()
```

 Output:

```
1
2
3
4
```

#### func (*Runner) [Restart](/runner.go#L21)

`func (r *Runner) Restart() error`

#### func (*Runner) [Start](/runner.go#L31)

`func (r *Runner) Start() error`

#### func (*Runner) [Stop](/runner.go#L69)

`func (r *Runner) Stop()`

Stop tries to kill the process group and waits for the command to finish.
If the process is not running it does not return any error

### type [Server](/tcpdevserver.go#L18)

`type Server struct { ... }`

#### func (*Server) [Run](/tcpdevserver.go#L65)

`func (s *Server) Run(main func(c *Runner) error) error`

## Sub Packages

* [app](./app)

* [app/test](./app/test)

* [test](./test)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)

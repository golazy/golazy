package lazyapp

import (
	"testing"
)

func TestStackDecode(t *testing.T) {
	lines := StackDecode(stackSample)

	expect := func(pack, f, file, line string) {
		for _, l := range lines {
			if l.Package == pack && l.Func == f && l.File == file && l.Line == line {
				return
			}
		}

		t.Errorf("not found: Package: %q Func: %q File: %q Line: %s", pack, f, file, line)
		for _, l := range lines {
			if (pack != "" && l.Package == pack) ||
				(f != "" && l.Func == f) ||
				(line != "" && l.Line == line) {
				t.Logf("    found: Package: %q Func: %q File: %q Line: %s", l.Package, l.Func, l.File, l.Line)
			}

		}
	}

	expect("golazy.dev/lazyaction", "func (*Dispatcher) dispatch(...)", "/home/guillermo/Projects/golazy/worktree/main/lazyaction/dispater_dispatch.go", "47")
	expect("net/http", "func HandlerFunc ServeHTTP(...)", "/usr/local/go/src/net/http/server.go", "2122")
	expect("net/http", "func (*Server) Serve(...)", "/usr/local/go/src/net/http/server.go", "3089")
	expect("github.com/felixge/httpsnoop", "func CaptureMetricsFn(...)", "/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go", "81")

}

var stackSample = []byte(`goroutine 9 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x65
golazy.dev/lazyapp.panicMiddleware.func1.1()
	/home/guillermo/Projects/golazy/worktree/main/lazyapp/panics.go:54 +0x159
panic({0x8c5a20, 0xc0004963b0})
	/usr/local/go/src/runtime/panic.go:884 +0x213
golazy.dev/lazyaction.(*Dispatcher).dispatch(0xd2f8c0, 0xc0002f2a00, {0xa366d0?, 0xc0004aa260}, 0xc000043100)
	/home/guillermo/Projects/golazy/worktree/main/lazyaction/dispater_dispatch.go:47 +0x9b9
golazy.dev/lazyaction.(*Dispatcher).ServeHTTP.func1({0xa366d0?, 0xc0004aa260?}, 0xc0004aa240?)
	/home/guillermo/Projects/golazy/worktree/main/lazyaction/dispatcher.go:146 +0x3b
net/http.HandlerFunc.ServeHTTP(0x40f487?, {0xa366d0?, 0xc0004aa260?}, 0x496301?)
	/usr/local/go/src/net/http/server.go:2122 +0x2f
github.com/felixge/httpsnoop.CaptureMetrics.func1({0xa366d0?, 0xc0004aa260?})
	/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go:30 +0x39
github.com/felixge/httpsnoop.CaptureMetricsFn({0xa366d0, 0xc0004aa1c0}, 0xc0000e74d8)
	/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go:81 +0x262
github.com/felixge/httpsnoop.CaptureMetrics({0xa33a80?, 0xc0004aa1e0?}, {0xa366d0?, 0xc0004aa1c0?}, 0x0?)
	/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go:29 +0x6b
golazy.dev/lazyaction.(*Dispatcher).ServeHTTP(0xd2f8c0, {0xa366d0, 0xc0004aa1c0}, 0xc000043100)
	/home/guillermo/Projects/golazy/worktree/main/lazyaction/dispatcher.go:149 +0x16a
golazy.dev/lazyassets.(*Assets).NewMiddleware.func1({0xa366d0, 0xc0004aa1c0}, 0x0?)
	/home/guillermo/Projects/golazy/worktree/main/lazyassets/manager.go:121 +0x92
net/http.HandlerFunc.ServeHTTP(0x0?, {0xa366d0?, 0xc0004aa1c0?}, 0x0?)
	/usr/local/go/src/net/http/server.go:2122 +0x2f
github.com/felixge/httpsnoop.CaptureMetrics.func1({0xa366d0?, 0xc0004aa1c0?})
	/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go:30 +0x39
github.com/felixge/httpsnoop.CaptureMetricsFn({0xa36040, 0xc00049e080}, 0xc0000e77d8)
	/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go:81 +0x262
github.com/felixge/httpsnoop.CaptureMetrics({0xa33a80?, 0xc000316120?}, {0xa36040?, 0xc00049e080?}, 0xa352e8?)
	/home/guillermo/go/pkg/mod/github.com/felixge/httpsnoop@v1.0.1/capture_metrics.go:29 +0x6b
golazy.dev/lazyapp.loggerMiddleware.func1({0xa36040, 0xc00049e080}, 0xc000043100)
	/home/guillermo/Projects/golazy/worktree/main/lazyapp/log.go:32 +0xbc
net/http.HandlerFunc.ServeHTTP(0xd2ee60?, {0xa36040?, 0xc00049e080?}, 0xc0004802a0?)
	/usr/local/go/src/net/http/server.go:2122 +0x2f
golazy.dev/lazyapp.panicMiddleware.func1({0xa36070?, 0xc0004802a0}, 0xc00049a101?)
	/home/guillermo/Projects/golazy/worktree/main/lazyapp/panics.go:58 +0xe9
net/http.HandlerFunc.ServeHTTP(0xc0000afa30?, {0xa36070?, 0xc0004802a0?}, 0xc0000afaa8?)
	/usr/local/go/src/net/http/server.go:2122 +0x2f
golazy.dev/lazydev/injector.New.func1.1({0xa362b0?, 0xc00049a0e0}, 0x0?)
	/home/guillermo/Projects/golazy/worktree/main/lazydev/injector/injector.go:16 +0xd1
net/http.HandlerFunc.ServeHTTP(0xc000496360?, {0xa362b0?, 0xc00049a0e0?}, 0x8eb960?)
	/usr/local/go/src/net/http/server.go:2122 +0x2f
golazy.dev/lazyapp.(*App).ServeHTTP(0x0?, {0xa362b0?, 0xc00049a0e0?}, 0x46618e?)
	/home/guillermo/Projects/golazy/worktree/main/lazyapp/app.go:88 +0x32
net/http.serverHandler.ServeHTTP({0xc000314e70?}, {0xa362b0, 0xc00049a0e0}, 0xc000043100)
	/usr/local/go/src/net/http/server.go:2936 +0x316
net/http.(*conn).serve(0xc00026b0e0, {0xa36b98, 0xc000314900})
	/usr/local/go/src/net/http/server.go:1995 +0x612
created by net/http.(*Server).Serve
	/usr/local/go/src/net/http/server.go:3089 +0x5ed`)

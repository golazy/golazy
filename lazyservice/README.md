# lazyservice

## Description

The `lazyservice` package provides a framework for building lazy applications in Go. A lazy application is an application that starts and stops services on demand. It allows you to define services as functions and run them within the application. The package provides an interface for defining services, adding values and types to the application, and running the application and its services. It also provides a default logger implementation and supports colored debug messages and JSON logs. The application uses trace regions for the app and each of the services.

## Usage

### Creating a Service

To create a service using `lazyservice`, you need to define a function that takes a context and a logger as parameters and returns an error. Here's an example:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golazy.dev/lazycontext"
	"golazy.dev/lazyservice"
)

func main() {
	service := func(ctx context.Context, l *slog.Logger) error {
		l.Info("hi")
		return fmt.Errorf("hi")
	}
	srv := lazyservice.ServiceFunc("basic", service)

	if srv.Desc().Name() != "basic" {
		fmt.Println(srv.Desc().Name())
	}

	err := srv.Run(lazycontext.New())
	if err.Error() != "hi" {
		fmt.Println("error didn't say hi")
	}
}
```

### Running the Application

You can create a new application using the `New` function and add services to it. The application can then be run with a context. Here's an example:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"golazy.dev/lazycontext"
	"golazy.dev/lazyservice"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	app := lazyservice.New()

	app.AddService(lazyservice.ServiceFunc("http", func(ctx context.Context, l *slog.Logger) error {
		s := &http.Server{
			Addr: ":8083",
		}

		idleConnsClosed := make(chan struct{})
		go func() {
			defer close(idleConnsClosed)
			<-ctx.Done()
			l.InfoContext(ctx, "shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err := s.Shutdown(ctx)
			if err == nil || err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			l.ErrorContext(ctx, err.Error(), "err", err)
		}()

		l.InfoContext(ctx, "listening on 8083")
		err := s.ListenAndServe()
		if err != http.ErrServerClosed {
			return err
		}
		<-idleConnsClosed
		return nil
	}))

	err := app.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
```

## Dependencies and Installation

To use `lazyservice`, you need to have Go installed on your system. You can install `lazyservice` using the following command:

```sh
go get golazy.dev/lazyservice
```

## Contributing and Reporting Issues

If you would like to contribute to the development of `lazyservice` or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.

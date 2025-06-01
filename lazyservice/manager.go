package lazyservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"runtime/trace"

	"github.com/golazy/golazy/lazycontext"
	"github.com/golazy/golazy/lazyenv"
	"github.com/lmittmann/tint"
	"golang.org/x/sync/errgroup"
)

type Manager interface {
	AddService(Service)
	Run(context.Context) error
}

// New creates a new app setting the name and version of the app.
// If the name is empty, it tries to use the base name of the executable.
// If the version is empty, it tries to use the modification time of the executable.
func New() Manager {
	return (&manager{}).init()
}

func (a *manager) init() *manager {
	if a.done == nil {
		a.done = make(chan struct{})
	}
	return a
}

type manager struct {
	logger   *slog.Logger
	services []Service
	done     chan struct{}
	cancel   context.CancelFunc
}

func (a *manager) AddService(s Service) {
	a.services = append(a.services, s)
}

// Run runs the app and all its services.
// If the app does not have a context, it creates a new one. This new context will be canceled when the app receives an interrupt signal.
// The app sets up a default slog Logger unless one is set through AppSet.
// The default logger outputs colored debug messages if the output is a terminal, otherwise it outputs JSON logs.
// The logger will have the app name and version as attributes for the json output.
// Each service will also have the service attribute set to the service name.
// If any service returns, all the contexts will be canceled and the app will wait for all services to stop.
// The application uses trace regions for the app and for each of the services.
func (a *manager) Run(ctx context.Context) error {
	//	if a.captureInt {
	//a.ctx, a.cancel = signal.NotifyContext(a.ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	//a.cancel()
	//}

	// Check if the context is a lazycontext
	lctx := lazycontext.Get[lazycontext.LazyContext](ctx)
	if lctx == nil {
		lctx = lazycontext.NewWithContext(ctx)
		ctx = lctx
	}

	// Setup a logger
	a.logger = setupLogger(ctx)
	go func() {
		if done := ctx.Done(); done != nil {
			<-done
			err := ctx.Err()
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				a.logger.Info("done")
			} else {
				a.logger.Error("done with error: " + ctx.Err().Error())
			}
		}
	}()

	appRegion := trace.StartRegion(ctx, "lazyapp.Run")
	defer appRegion.End()

	a.logger.Info("starting app")

	grp, grpCtx := errgroup.WithContext(ctx)

	for _, s := range a.services {
		s := s // create a local variable and assign the value of s to it
		grp.Go(func() error {
			l := a.logger.With("service", s.Desc().Name())
			ctx := context.WithValue(grpCtx, reflect.TypeOf(a.logger), l)

			srvReg := trace.StartRegion(ctx, "service:"+s.Desc().Name())
			defer srvReg.End()

			l.InfoContext(ctx, "starting service")

			err := s.Run(ctx)
			if errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded) {
				l.InfoContext(ctx, "stopped")
				return nil
			}
			if err != nil {
				l.ErrorContext(ctx, err.Error())
			}
			l.InfoContext(ctx, "app gracefully stoped")
			return err
		})
	}
	return grp.Wait()
}

func setupLogger(ctx context.Context) *slog.Logger {
	logger := lazycontext.Get[*slog.Logger](ctx)
	if logger != nil {
		return logger
	}
	if lazyenv.IsDevelopment() {
		logger = slog.New(tint.NewHandler(os.Stdout,
			&tint.Options{
				AddSource: true,
				Level:     slog.LevelDebug,
				ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
					switch string(attr.Key) {
					case "app", "version":
						return attr
					}
					return attr
				},
			},
		))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelWarn,
		}))

	}
	lazycontext.Set(ctx, logger)
	return logger
}

func getVersion() string {
	s, err := os.Stat(os.Args[0])
	if err != nil {
		return ""
	}
	return s.ModTime().Format("20060102-150405")

}

package lazyapp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golazy.dev/lazyaction"
	"golazy.dev/lazyview/component"
	"golazy.dev/lazyview/static_files"
)

type App struct {
	Name        string
	Router      lazyaction.Dispatcher
	Server      http.Server
	Files       *static_files.Manager
	mounts      mounts
	MiddleWares []Middleware
	h           http.Handler
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}

func (a *App) Route(args ...any) {
	a.Router.Route(args...)
}

func (a *App) Resource(resource any, opts ...*lazyaction.ResourceOptions) {
	a.Router.Resource(resource, opts...)
}

func (a *App) Init() {
	a.Router.Files = a.Files

	a.h = &a.Router

	// Add mounts
	if a.mounts != nil {
		a.h = a.mounts.Middleware(a.h)
	}

	// Add files
	if a.Files != nil {
		a.h = a.Files.NewMiddleware(a.h)
	}

	// Add logger
	a.h = loggerMiddleware(a.h)

	// Add panic handler
	a.h = panicMiddleware(a.h)

	// Add middlewares
	for _, m := range a.MiddleWares {
		a.h = m(a.h)
	}
}

func (a *App) Boot() {
	a.Init()

	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	command := &cobra.Command{
		Use: a.Name,
		Run: func(c *cobra.Command, args []string) {

			addr := viper.GetString("port")
			if !strings.Contains(addr, ":") {
				addr = ":" + addr
			}

			l, err := getListener(addr)
			if err != nil {
				panic(err)
			}
			err = http.Serve(l, a)
			if err != nil {
				panic(err)
			}

		},
	}
	command.AddCommand(&cobra.Command{
		Use: "routes",
		Run: func(c *cobra.Command, args []string) {
			fmt.Println(a.Router)
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "assets",
		Short: "Install all the assets",
		Run: func(c *cobra.Command, args []string) {
			component.InstallAll(component.InstallOptions{
				Path:  "assets/public",
				Cache: "assets/cache",
			})

		},
	})

	command.PersistentFlags().StringP("port", "p", "localhost:2000", "Listening port or address")
	viper.BindPFlag("port", command.PersistentFlags().Lookup("port"))
	viper.BindEnv("port")

	command.Execute()
}

func (a *App) Mount(path string, h http.Handler) {
	if a.mounts == nil {
		a.mounts = make(map[string]http.Handler)
	}
	a.mounts[path] = h
}

func getListener(addr string) (net.Listener, error) {
	if strings.HasPrefix(addr, "fd:") {
		fd, err := strconv.Atoi(addr[3:])
		if err != nil {
			return nil, err
		}
		listenerFile := os.NewFile(uintptr(fd), "listener")
		if listenerFile == nil {
			return nil, fmt.Errorf("expecting listener in FD %d", fd)
		}

		l, err := net.FileListener(listenerFile)
		if err != nil {
			return nil, fmt.Errorf("error creating listener: %s", err)
		}
		return l, nil
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return net.ListenTCP("tcp", tcpAddr)
}

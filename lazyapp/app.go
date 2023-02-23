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
	"golazy.dev/lazyview/asset_manager"
)

// type App struct {
// 	LayoutManager layout_manager.LayoutManager
// 	AssetManager  asset_manager.asset_manager
// 	Router        lazyaction.Router
// }

type App struct {
	Name         string
	Router       lazyaction.Routes
	Server       http.Server
	AssetManager *asset_manager.AssetManager
	MiddleWares  []Middleware
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}

func (a *App) initialize() {
	if a.AssetManager == nil {
		a.Router = lazyaction.Routes{}
	}

}

func (a *App) Boot() {
	a.initialize()

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
			err = http.Serve(l, handler{a})
			if err != nil {
				panic(err)
			}

		},
	}

	command.PersistentFlags().StringP("port", "p", "localhost:2000", "Listening port or address")
	viper.BindPFlag("port", command.PersistentFlags().Lookup("port"))
	viper.BindEnv("port")

	command.Execute()
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

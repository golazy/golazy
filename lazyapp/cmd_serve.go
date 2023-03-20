package lazyapp

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = func(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use: a.Name,
		Run: func(c *cobra.Command, args []string) {

			addr := viper.GetString("port")
			if !strings.Contains(addr, ":") {
				addr = ":" + addr
			}

			l, err := getListener(a, addr)
			if err != nil {
				panic(err)
			}
			fmt.Println("Listening on", l.Addr().String())
			err = http.Serve(l, a)
			if err != nil {
				panic(err)
			}

		},
	}

	cmd.Flags().StringP("port", "p", "127.0.0.1:2000", "Port to listen on")
	viper.BindPFlag("port", cmd.Flags().Lookup("port"))
	viper.BindEnv("port")

	return cmd
}

func getListener(a *App, addr string) (net.Listener, error) {
	if a.Addr != "" {
		addr = a.Addr
	}
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

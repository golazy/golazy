package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"flag"
)

var listenAddr string
var helpWanted bool

func init() {
	flag.StringVar(&listenAddr, "listen", "localhost:8080", "Listen address fd:/1 or host:port")
	flag.BoolVar(&helpWanted, "h", false, "Show help")
}

func getListener() (net.Listener, error) {
	if strings.HasPrefix(listenAddr, "fd:") {
		fd, err := strconv.Atoi(listenAddr[3:])
		if err != nil {
			return nil, err
		}
		listenerFile := os.NewFile(uintptr(fd), "listener")
		if listenerFile == nil {
			return nil, fmt.Errorf("Expecting listener in FD %d", fd)
		}

		l, err := net.FileListener(listenerFile)
		if err != nil {
			return nil, fmt.Errorf("Error creating listener: %s", err)
		}
		return l, nil
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return net.ListenTCP("tcp", tcpAddr)
}

func main() {
	if helpWanted {
		flag.Usage()
		os.Exit(0)
	}

	l, err := getListener()
	if err != nil {
		panic(err)
	}

	httpServer := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello world"))
		}),
	}

	fmt.Println("Listening on", listenAddr)

	err = httpServer.Serve(l)
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}

}

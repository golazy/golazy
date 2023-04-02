package dev

import (
	"fmt"
	"os"
	"os/signal"
	"portal/apps/portal"
	"strings"

	"golazy.dev/lazydev/cli/subargs"
	"golazy.dev/lazydev/devserver"
	"golazy.dev/lazydev/devserver/events"
	"golazy.dev/lazydev/portalserver"
)

type DevOpts struct {
	MainDir string
	Dir     string

	Portal bool
	Port   string
}

func runNoPortal(opts DevOpts) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	srv := devserver.New(devserver.Options{
		BuildDir:  opts.MainDir,
		RootDir:   opts.Dir,
		BuildArgs: strings.Split("-buildvcs=false", " "),
		RunEnv:    []string{"PORT=" + opts.Port},
		RunArgs:   subargs.Args,
		Events: func(e events.Event) {

			if e, ok := e.(events.Stdout); ok {
				os.Stdout.Write([]byte(e))
				return
			}

			if e, ok := e.(events.Stderr); ok {
				os.Stderr.Write([]byte(e))
				return
			}

			if e, ok := e.(events.BuildError); ok {
				fmt.Println(string(e.Out))
				return
			}

			fmt.Printf("#> %-15s %s\n", e.Type(), e.String())
		},
	})
	go func() {
		<-interrupt
		fmt.Println("Got CTRL+C, shutting down...")
		srv.Close()
	}()

	return srv.Serve()
}

func runPortal(opts DevOpts) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	srv := portalserver.New(portalserver.Options{
		Addr:      opts.Port,
		BuildDir:  opts.MainDir,
		BuildArgs: strings.Split("-buildvcs=false", " "),
		RunArgs:   subargs.Args,
		App:       portal.App,
	})

	go func() {
		<-interrupt
		fmt.Println("Got CTRL+C, shutting down...")
		srv.Close()
	}()

	return srv.ListenAndServe()
}
func Run(opts DevOpts) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if opts.Portal {
		return runPortal(opts)
	}

	return runNoPortal(opts)
}

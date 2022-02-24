package lazydev

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dietsche/rfsnotify"
)

type msg int

const (
	childStartEvent msg = iota
	childStopEvent
	fileChangeEvent
	timeoutEvent
)

var messages = make(chan (msg))

func parentStart() {
	go parentWatchChanges()
	go parentStartChild()

	started := false
	var timer *time.Timer
	for {
		switch <-messages {
		case childStartEvent:
			started = true
		case childStopEvent:
			started = false
		case fileChangeEvent:
			if timer == nil {
				timer = time.AfterFunc(1000*time.Millisecond, func() { messages <- timeoutEvent })
			} else {
				timer.Reset(1000 * time.Millisecond)
			}
		case timeoutEvent:
			log.Println("Timeout")
			if !started {
				continue
			}
			parentKillChild()
			go parentStartChild()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

var WatchPaths = []string{"."}

func init() {
	paths := os.Getenv("LAZYWATCH")
	if paths == "" {
		return
	}
	WatchPaths = strings.Split(paths, ",")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			parentKillChild()
			os.Exit(0)
		}
	}()

}

func parentWatchChanges() {
	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				messages <- fileChangeEvent
				fmt.Println(event.Name)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	for _, s := range WatchPaths {
		if strings.HasSuffix(s, "/...") {
			err = watcher.AddRecursive(s[:len(s)-4])
		} else {
			err = watcher.Add(s)
		}
		if err != nil {
			log.Fatal("Can't watch " + s + ": " + err.Error())
		}
	}

	<-done
}

var childCmd *exec.Cmd

func parentStartChild() {
	fmt.Println("Starting")
	files, err := filepath.Glob("*.go")
	if err != nil {
		log.Fatal(err)
	}

	args := []string{"run"}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		args = append(args, file)
	}

	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LAZYDEVCHILD=true")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	f, err := listener.File()
	if err != nil {
		log.Fatal(err)
	}
	cmd.ExtraFiles = []*os.File{f}
	childCmd = cmd
	// This is a race condition
	messages <- childStartEvent
	cmd.Run()
	log.Println("And the child is death", cmd.ProcessState)

}

func parentKillChild() {
	syscall.Kill(-childCmd.Process.Pid, syscall.SIGKILL)
	childCmd.Process.Wait()
	log.Println("Child killed")
}

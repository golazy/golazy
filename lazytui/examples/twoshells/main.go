package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golazy.dev/lazytui/encoding/tty"
	"golazy.dev/lazytui/pty"
	"golazy.dev/lazytui/window"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("twoshells", flag.ContinueOnError)
	flags.SetOutput(stderr)

	rows := flags.Int("rows", 24, "total terminal rows")
	cols := flags.Int("cols", 80, "total terminal columns")
	paddingCols := flags.Int("pad-x", 2, "horizontal padding cells")
	paddingRows := flags.Int("pad-y", 1, "vertical padding cells")
	probe := flags.Bool("probe", false, "run stty size in both panes and exit")
	shellFlag := flags.String("shell", "", "shell path")
	if err := flags.Parse(args); err != nil {
		return err
	}

	size := tty.Size{Rows: *rows, Cols: *cols}
	panes, err := window.SplitSideBySide(size, *paddingCols, *paddingRows)
	if err != nil {
		return err
	}

	shell, err := resolveShell(*shellFlag)
	if err != nil {
		return err
	}

	if *probe {
		return runProbe(stdout, shell, panes)
	}
	return runShells(stdout, shell, panes)
}

func runProbe(stdout io.Writer, shell string, panes [2]window.Rect) error {
	left, err := runSized(shell, panes[0].Size(), "stty size")
	if err != nil {
		return fmt.Errorf("left pane: %w", err)
	}
	right, err := runSized(shell, panes[1].Size(), "stty size")
	if err != nil {
		return fmt.Errorf("right pane: %w", err)
	}

	fmt.Fprintf(stdout, "left %s\n", strings.Join(strings.Fields(left), " "))
	fmt.Fprintf(stdout, "right %s\n", strings.Join(strings.Fields(right), " "))
	return nil
}

func runShells(stdout io.Writer, shell string, panes [2]window.Rect) error {
	fmt.Fprintf(stdout, "left pane: %dx%d at row %d col %d\n", panes[0].Rows, panes[0].Cols, panes[0].Row, panes[0].Col)
	fmt.Fprintf(stdout, "right pane: %dx%d at row %d col %d\n", panes[1].Rows, panes[1].Cols, panes[1].Row, panes[1].Col)
	fmt.Fprintln(stdout, "run with -probe to verify child PTY sizes; interactive pane rendering is the next layer")

	left := pty.Command(shell, "-i")
	left.Size = panes[0].Size()
	left.Env = append(os.Environ(), "TERM=xterm-256color")
	right := pty.Command(shell, "-i")
	right.Size = panes[1].Size()
	right.Env = append(os.Environ(), "TERM=xterm-256color")

	if err := left.Start(); err != nil {
		return fmt.Errorf("left pane: %w", err)
	}
	defer left.Process().Kill()
	if err := right.Start(); err != nil {
		_ = left.Process().Kill()
		return fmt.Errorf("right pane: %w", err)
	}
	defer right.Process().Kill()
	return errors.New("interactive pane rendering is not implemented yet")
}

func runSized(shell string, size tty.Size, script string) (string, error) {
	var out bytes.Buffer
	cmd := pty.Command(shell, "-c", script)
	cmd.Size = size
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func resolveShell(value string) (string, error) {
	if value != "" {
		return value, nil
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell, nil
	}
	shell, err := exec.LookPath("sh")
	if err != nil {
		return "", err
	}
	return shell, nil
}

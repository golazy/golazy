package progress

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"unicode"
)

// Cmd returns a task function that runs a command.
//
// When args is empty, command is split into argv using simple shell-like
// whitespace, quote, and backslash handling. When args is not empty, command is
// used as argv[0] without splitting.
func Cmd(command string, args ...string) Func {
	return commandFunc(false, command, args...)
}

// CmdWarn is like Cmd, but returns Warn for command failures.
func CmdWarn(command string, args ...string) Func {
	return commandFunc(true, command, args...)
}

// Mise returns a task function that runs a command through mise.
//
// The generated command is mise exec --raw -- <command> <args...>.
func Mise(command string, args ...string) Func {
	return miseFunc(false, command, args...)
}

// MiseWarn is like Mise, but returns Warn for command failures.
func MiseWarn(command string, args ...string) Func {
	return miseFunc(true, command, args...)
}

func commandFunc(warn bool, command string, args ...string) Func {
	return func(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		argv, err := commandArgs(command, args...)
		if err == nil {
			err = runProgram(stdin, stdout, stderr, argv[0], argv[1:])
		}
		if err != nil && warn && !isWarning(err) {
			return Warn{Err: err}
		}
		return err
	}
}

func miseFunc(warn bool, command string, args ...string) Func {
	return func(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		argv, err := commandArgs(command, args...)
		if err == nil {
			miseArgs := append([]string{"exec", "--raw", "--"}, argv...)
			err = runProgram(stdin, stdout, stderr, "mise", miseArgs)
		}
		if err != nil && warn && !isWarning(err) {
			return Warn{Err: err}
		}
		return err
	}
}

func commandArgs(command string, args ...string) ([]string, error) {
	if len(args) != 0 {
		if strings.TrimSpace(command) == "" {
			return nil, fmt.Errorf("command is empty")
		}
		return append([]string{command}, args...), nil
	}
	argv, err := splitCommandLine(command)
	if err != nil {
		return nil, err
	}
	if len(argv) == 0 {
		return nil, fmt.Errorf("command is empty")
	}
	return argv, nil
}

var runProgram = func(stdin io.Reader, stdout io.Writer, stderr io.Writer, command string, args []string) error {
	process := exec.Command(command, args...)
	process.Stdin = stdin
	process.Stdout = stdout
	process.Stderr = stderr
	if err := process.Run(); err != nil {
		return fmt.Errorf("%s: %w", formatCommand(command, args), err)
	}
	return nil
}

func formatCommand(command string, args []string) string {
	parts := append([]string{command}, args...)
	return strings.Join(parts, " ")
}

func splitCommandLine(value string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	haveToken := false

	for _, char := range value {
		if escaped {
			current.WriteRune(char)
			haveToken = true
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			haveToken = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
				continue
			}
			current.WriteRune(char)
			haveToken = true
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			haveToken = true
			continue
		}
		if unicode.IsSpace(char) {
			if haveToken {
				args = append(args, current.String())
				current.Reset()
				haveToken = false
			}
			continue
		}
		current.WriteRune(char)
		haveToken = true
	}

	if escaped {
		return nil, fmt.Errorf("split command: unfinished escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("split command: unterminated quote")
	}
	if haveToken {
		args = append(args, current.String())
	}
	return args, nil
}

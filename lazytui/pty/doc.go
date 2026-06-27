// Package pty starts commands behind pseudo terminals.
//
// It is the process side of Lazy Terminal's fake terminal work: callers assign
// cell sizes to a child process, and the backend exposes those sizes through
// the operating system's terminal ioctl layer.
package pty

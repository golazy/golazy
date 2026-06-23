// Package progress runs named terminal tasks with compact status output.
//
// Tasks receive standard input, output, and error streams. Output produced by a
// task is captured while the task runs and is only printed if the task returns
// an error or warning.
package progress

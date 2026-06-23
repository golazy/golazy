// Package progress runs named terminal tasks with compact status output.
//
// Tasks receive standard input, output, and error streams. Output produced by a
// task is captured while the task runs and is only printed if the task returns
// an error or warning. UITask gives a sequential task temporary direct terminal
// control through UI.Takeover for prompts, diffs, or other interactive work.
package progress

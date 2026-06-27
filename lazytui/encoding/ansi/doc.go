// Package ansi encodes and decodes ANSI and VT terminal byte streams.
//
// The package is deliberately protocol-level. It turns byte streams into
// tokens and tokens or operations into byte streams, but it does not mutate a
// terminal screen. Terminal emulation belongs in higher-level lazytui packages.
package ansi

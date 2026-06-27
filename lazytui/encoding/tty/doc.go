// Package tty encodes terminal-control requests and applies them to terminal
// devices.
//
// The package separates the client that asks for terminal operations from the
// server that owns the real terminal backend. That split lets future GoLazy
// processes attach to an existing terminal supervisor without sharing raw OS
// handles in the public API.
package tty

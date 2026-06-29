// Package lazyconfig fills configuration structs from environment variables.
//
// It is a standalone loader: it reads the current process environment through
// os.Environ, matches exported struct fields to environment names, parses
// values into a new config value, and returns that value to the caller. It does
// not depend on a GoLazy application and does not keep package-level state.
//
// Use Getenv with an application-owned struct:
//
//	type Config struct {
//		Port       int
//		ListenAddr string `var:"LISTEN_ADDR" default:":3000"`
//	}
//
//	config, err := lazyconfig.Getenv[Config]()
//
// Use MustGetenv when invalid environment values should fail during package
// initialization. The lazy CLI uses this shape for its package-level Config
// singleton, and lazyapp uses it internally for app process settings such as
// ADDR, PORT, and CONTROL_PLANE_ADDR before building the runtime server:
//
//	var Config = lazyconfig.MustGetenv[Config]()
//
// Field names are converted from CamelCase to upper snake case. For example,
// ListenAddr reads LISTEN_ADDR, and compact fallback names are also accepted
// for acronym-heavy fields such as GoWork reading GOWORK. Use a var tag when
// the environment name is not derived from the field name, a default tag when a
// missing variable should use a fallback value, and a required or require tag
// when a missing or empty variable should return an error.
//
// Supported scalar field types are string, bool, signed and unsigned integers,
// floats, time.Duration, and types that implement encoding.TextUnmarshaler.
// Pointers to scalar values are allocated only when the matching value is
// present. Nested structs are loaded with the parent field name as a prefix.
// Slices can be loaded from indexed variables such as LISTENER_1_NAME and
// LISTENER_2_NAME; []string can also be loaded from a single comma- or
// whitespace-delimited value. When a loaded config value implements
// Validate() error, Getenv calls it before returning.
//
// Options can adjust environment name matching. RemoveEnvNamePrefix adds
// unprefixed aliases for environment variables, which is useful when a struct
// models a namespaced environment convention. lazytelemetry uses this option to
// read OpenTelemetry variables with their OTEL_ prefix while keeping its config
// struct names focused on the protocol fields:
//
//	type OTELConfig struct {
//		SDKDisabled bool
//		ServiceName string
//	}
//
//	config := lazyconfig.MustGetenv[OTELConfig](
//		lazyconfig.RemoveEnvNamePrefix("OTEL"),
//	)
//
// Values are trimmed before parsing. String, numeric, bool, pointer scalar, and
// []string fields are supported; []string values split on commas or whitespace.
// Bool fields treat yes, true, and 1 as true; no, false, 0, and unknown values
// are false. Bool matching is case-insensitive.
package lazyconfig

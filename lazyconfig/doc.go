// Package lazyconfig fills configuration structs from environment variables.
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
// initialization:
//
//	var Config = lazyconfig.MustGetenv[Config]()
//
// Options can adjust environment name matching. RemoveEnvNamePrefix adds
// unprefixed aliases for environment variables, which is useful when a struct
// models a namespaced environment convention:
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

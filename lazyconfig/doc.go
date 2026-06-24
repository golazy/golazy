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
package lazyconfig

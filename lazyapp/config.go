package lazyapp

import (
	"strconv"

	"golazy.dev/lazyconfig"
)

const defaultListenAddr = "127.0.0.1:3000"

type environmentConfig struct {
	Addr             string `default:"127.0.0.1:3000"`
	Port             int    `default:"0"`
	ControlPlaneAddr string
	LazyappMigrate   string         `var:"LAZYAPP_MIGRATE"`
	LazyappCacheSize cacheSizeBytes `var:"LAZYAPP_CACHE_SIZE"`
}

var environment = loadEnvironment()

func loadEnvironment() environmentConfig {
	return lazyconfig.MustGetenv[environmentConfig]()
}

func listenAddr() string {
	normalizedAddr := normalizeListenAddr(environment.Addr)
	if environment.Port != 0 && (normalizedAddr == "" || normalizedAddr == defaultListenAddr) {
		return ":" + strconv.Itoa(environment.Port)
	}
	return normalizedAddr
}

func controlPlaneListenAddr() (string, bool) {
	if environment.ControlPlaneAddr == "" {
		return "", false
	}
	return normalizeListenAddr(environment.ControlPlaneAddr), true
}

func normalizeListenAddr(addr string) string {
	if _, err := strconv.ParseUint(addr, 10, 16); err == nil {
		return ":" + addr
	}
	return addr
}

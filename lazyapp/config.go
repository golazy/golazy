package lazyapp

import (
	"fmt"
	"strconv"

	"golazy.dev/lazyconfig"
)

type environmentConfig struct {
	Addr             string `var:"ADDR"`
	Port             string `var:"PORT"`
	ControlPlaneAddr string `var:"CONTROL_PLANE_ADDR"`
}

func loadEnvironmentConfig() environmentConfig {
	config, err := lazyconfig.Getenv[environmentConfig]()
	if err != nil {
		panic(fmt.Errorf("lazyapp: read environment config: %w", err))
	}
	return config
}

func listenAddr() string {
	config := loadEnvironmentConfig()
	if config.Addr != "" {
		return normalizeListenAddr(config.Addr)
	}
	if config.Port != "" {
		return normalizeListenAddr(config.Port)
	}
	return ":3000"
}

func controlPlaneListenAddr() (string, bool) {
	config := loadEnvironmentConfig()
	if config.ControlPlaneAddr == "" {
		return "", false
	}
	return normalizeListenAddr(config.ControlPlaneAddr), true
}

func normalizeListenAddr(addr string) string {
	if _, err := strconv.ParseUint(addr, 10, 16); err == nil {
		return ":" + addr
	}
	return addr
}

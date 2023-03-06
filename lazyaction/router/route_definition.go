package router

import (
	"net/url"
	"strings"
)

type RouteDefinition struct {
	Method string
	Port   string
	Domain string
	Scheme string
	Path   string
}

func NewRouteDefinition(def string) *RouteDefinition {
	method := "GET"
	if strings.Contains(def, " ") {
		parts := strings.Split(def, " ")
		method = parts[0]
		def = parts[1]
	}
	u, err := url.Parse(def)
	if err != nil {
		panic(err)
	}

	return &RouteDefinition{
		Method: method,
		Port:   u.Port(),
		Domain: u.Hostname(),
		Scheme: u.Scheme,
		Path:   u.Path,
	}
}

package routertest

import (
	"errors"
	"fmt"

	"golazy.dev/lazyaction/internal/router"
)

func ExpectRoute(routes []*router.Route, expected *router.Route) error {
	if expected.Verb == "" {
		expected.Verb = "GET"
	}
	r := FindRoute(routes, expected)
	if r == nil {
		return errors.New("Route not found: " + expected.Verb + " " + expected.Path)
	}
	if err := CompareRoute(r, expected); err != nil {
		return err
	}
	return nil
}

func FindRoute(routes []*router.Route, expected *router.Route) *router.Route {
	for _, route := range routes {
		if route.Path == expected.Path && route.Verb == expected.Verb {
			return route
		}
	}
	return nil
}

func CompareRoute(original, expected *router.Route) error {
	var errs []error
	if original == nil || expected == nil {
		return fmt.Errorf("missing routes to compare")
	}

	if expected.Verb != "" {
		if original.Verb != expected.Verb {
			errs = append(errs, fmt.Errorf("expected verb %s, got %s", expected.Verb, original.Verb))
		}
	} else {
		if original.Verb != "GET" {
			errs = append(errs, fmt.Errorf("expected empty verb to generate GET, got %s", original.Verb))
		}
	}

	if expected.Path != "" {
		if original.Path != expected.Path {
			errs = append(errs, fmt.Errorf("expected path %s, got %s", expected.Path, original.Path))
		}
	}

	if expected.Name != "" {
		if original.Name != expected.Name {
			errs = append(errs, fmt.Errorf("expected name %s, got %s", expected.Name, original.Name))
		}
	}

	if expected.Args != nil {
		if len(original.Args) != len(expected.Args) {
			errs = append(errs, fmt.Errorf("expected %v arguments, got %v", expected.Args, original.Args))
		}

		for i, arg := range expected.Args {
			if original.Args[i] != arg {
				errs = append(errs, fmt.Errorf("expected argument %s, got %s", arg, original.Args[i]))
			}
		}
	}

	if expected.Rets != nil {
		if len(original.Rets) != len(expected.Rets) {
			errs = append(errs, fmt.Errorf("expected %d return values, got %d", len(expected.Rets), len(original.Rets)))
		}

		for i, ret := range expected.Rets {
			if original.Rets[i] != ret {
				errs = append(errs, fmt.Errorf("expected return value %s, got %s", ret, original.Rets[i]))
			}
		}
	}

	if expected.Controller != nil {
		if original.Controller != expected.Controller {
			errs = append(errs, fmt.Errorf("expected controller %s, got %s", expected.Controller, original.Controller))
		}
	}

	if expected.ControllerName != "" {
		if original.ControllerName != expected.ControllerName {
			errs = append(errs, fmt.Errorf("expected controller name %s, got %s", expected.ControllerName, original.ControllerName))
		}
	}

	if expected.Plural != "" {
		if original.Plural != expected.Plural {
			errs = append(errs, fmt.Errorf("expected plural %s, got %s", expected.Plural, original.Plural))
		}
	}

	if expected.Singular != "" {
		if original.Singular != expected.Singular {
			errs = append(errs, fmt.Errorf("expected singular %s, got %s", expected.Singular, original.Singular))
		}
	}

	if expected.ParamName != "" {
		if original.ParamName != expected.ParamName {
			errs = append(errs, fmt.Errorf("expected param name %s, got %s", expected.ParamName, original.ParamName))
		}
	}

	if expected.Name != "" {
		if original.Name != expected.Name {
			errs = append(errs, fmt.Errorf("expected name %s, got %s", expected.Name, original.Name))
		}
	}

	return errors.Join(errs...)
}

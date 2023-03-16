package lazyaction

import (
	"errors"
	"fmt"
	"reflect"
)

func ExpectAction(actions []*Action, expected *Action) error {
	if expected.Verb == "" {
		expected.Verb = "GET"
	}
	r := FindAction(actions, expected)
	if r == nil {
		return errors.New("Action not found: " + expected.Verb + " " + expected.URL.String())
	}
	if err := CompareAction(r, expected); err != nil {
		return err
	}
	return nil
}

func FindAction(actions []*Action, expected *Action) *Action {
	for _, action := range actions {
		if action.URL == expected.URL && action.Verb == expected.Verb {
			return action
		}
	}
	return nil
}

func CompareAction(original, expected *Action) error {
	var errs []error
	if original == nil || expected == nil {
		return fmt.Errorf("missing actions to compare")
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

	if expected.URL.String() != "" {
		if original.URL != expected.URL {
			errs = append(errs, fmt.Errorf("expected path %s, got %s", expected.URL.String(), original.URL.String()))
		}
	}

	if expected.Name != "" {
		if original.Name != expected.Name {
			errs = append(errs, fmt.Errorf("expected name %s, got %s", expected.Name, original.Name))
		}
	}
	if expected.ins != nil {
		if !reflect.DeepEqual(original.ins, expected.ins) {
			errs = append(errs, fmt.Errorf("expected argument %s, got %s", expected.ins, original.ins))
		}
	}
	if expected.outs != nil {
		if len(original.outs) != len(expected.outs) {
			errs = append(errs, fmt.Errorf("expected %d return values, got %d", len(expected.outs), len(original.outs)))
		}

		for i, ret := range expected.outs {
			if original.outs[i] != ret {
				errs = append(errs, fmt.Errorf("expected return value %s, got %s", ret, original.outs[i]))
			}
		}
	}
	if expected.fn != nil {
		if original.fn == nil {
			errs = append(errs, fmt.Errorf("expected function %s, got nil", expected.fn))
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

package lazyerrors

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golazy.dev/lazyschema"
)

const (
	ValidationPresence = "presence"
	ValidationMin      = "min"
	ValidationMax      = "max"
)

var timeType = reflect.TypeFor[time.Time]()

type customValidator interface {
	Validate() error
}

// ValidationError is one field validation failure.
type ValidationError struct {
	Type  string
	Field string
	Err   error
}

// NewValidationError creates one field validation failure.
func NewValidationError(field string, typ string, err error) ValidationError {
	return ValidationError{Field: field, Type: typ, Err: err}
}

func (e ValidationError) Error() string {
	if e.Err == nil {
		if e.Field == "" {
			return "validation failed"
		}
		return e.Field + ": validation failed"
	}
	if e.Field == "" {
		return e.Err.Error()
	}
	return e.Field + ": " + e.Err.Error()
}

func (e ValidationError) String() string {
	return e.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

// PresenceErr reports a missing value.
type PresenceErr struct{}

func (PresenceErr) Error() string {
	return "must be present"
}

// MinSizeErr reports a value smaller than the configured minimum size.
type MinSizeErr struct {
	Min     int
	Current int
}

func (e MinSizeErr) Error() string {
	return fmt.Sprintf("must have size at least %d", e.Min)
}

// MaxSizeErr reports a value larger than the configured maximum size.
type MaxSizeErr struct {
	Max     int
	Current int
}

func (e MaxSizeErr) Error() string {
	return fmt.Sprintf("must have size at most %d", e.Max)
}

// Validator validates struct fields with validate tags and joins any
// ValidationError values into one ordinary error. If the value implements
// Validate() error, Validator joins that returned error as well.
func Validator(value any) error {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return fmt.Errorf("lazyerrors: validator requires a struct")
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return fmt.Errorf("lazyerrors: validator received nil %s", rv.Type())
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("lazyerrors: validator requires a struct, got %s", rv.Type())
	}

	var out []error
	out = append(out, validateStruct(value, rv, nil)...)
	if custom, ok := value.(customValidator); ok {
		if err := custom.Validate(); err != nil {
			out = append(out, err)
		}
	}
	return errors.Join(out...)
}

// ValidationErrors extracts all ValidationError leaves from err.
func ValidationErrors(err error) []ValidationError {
	var out []ValidationError
	collectValidationErrors(err, &out)
	return out
}

// ErrorsFor groups validation errors by field.
func ErrorsFor(err error) map[string][]ValidationError {
	grouped := map[string][]ValidationError{}
	for _, validation := range ValidationErrors(err) {
		grouped[validation.Field] = append(grouped[validation.Field], validation)
	}
	return grouped
}

// FieldErrorsFor returns validation errors for field.
func FieldErrorsFor(err error, field string) []ValidationError {
	if strings.TrimSpace(field) == "" {
		return nil
	}
	return append([]ValidationError(nil), ErrorsFor(err)[field]...)
}

// Helpers returns template helpers for grouped validation errors.
func Helpers() map[string]any {
	return map[string]any{
		"errors_for": func(value any) map[string][]ValidationError {
			return ErrorsFor(validationErrorInput(value))
		},
		"field_errors_for": func(value any, field string) []ValidationError {
			return FieldErrorsFor(validationErrorInput(value), field)
		},
	}
}

func validationErrorInput(value any) error {
	if value == nil {
		return nil
	}
	err, _ := value.(error)
	return err
}

func validateStruct(root any, value reflect.Value, path []string) []error {
	var out []error
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" || field.Tag.Get("schema") == "-" || field.Tag.Get("form") == "-" {
			continue
		}
		if field.Tag.Get("validate") == "-" {
			continue
		}
		fieldValue := value.Field(i)
		fieldPath := append(append([]string(nil), path...), field.Name)
		rules, err := parseValidationRules(field.Tag.Get("validate"))
		if err != nil {
			out = append(out, err)
			continue
		}
		if len(rules) > 0 {
			out = append(out, validateField(root, fieldPath, fieldValue, rules)...)
		}
		if shouldRecurseValidation(fieldValue) {
			out = append(out, validateStruct(root, indirectValue(fieldValue), fieldPath)...)
		}
	}
	return out
}

func validateField(root any, fieldPath []string, value reflect.Value, rules []validationRule) []error {
	field := validationFieldName(root, fieldPath)
	var out []error
	for _, rule := range rules {
		switch rule.kind {
		case ValidationPresence:
			if !validationPresent(value) {
				out = append(out, ValidationError{Type: ValidationPresence, Field: field, Err: PresenceErr{}})
			}
		case ValidationMin:
			current, ok := validationSize(value)
			if !ok {
				out = append(out, fmt.Errorf("lazyerrors: min validation requires a sized field %s", strings.Join(fieldPath, ".")))
				continue
			}
			if current < rule.limit {
				out = append(out, ValidationError{
					Type:  ValidationMin,
					Field: field,
					Err:   MinSizeErr{Min: rule.limit, Current: current},
				})
			}
		case ValidationMax:
			current, ok := validationSize(value)
			if !ok {
				out = append(out, fmt.Errorf("lazyerrors: max validation requires a sized field %s", strings.Join(fieldPath, ".")))
				continue
			}
			if current > rule.limit {
				out = append(out, ValidationError{
					Type:  ValidationMax,
					Field: field,
					Err:   MaxSizeErr{Max: rule.limit, Current: current},
				})
			}
		}
	}
	return out
}

type validationRule struct {
	kind  string
	limit int
}

func parseValidationRules(tag string) ([]validationRule, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" || tag == "-" {
		return nil, nil
	}
	tag = strings.TrimPrefix(tag, "validate:")
	parts := strings.Split(tag, ";")
	rules := make([]validationRule, 0, len(parts))
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		switch {
		case part == ValidationPresence:
			rules = append(rules, validationRule{kind: ValidationPresence})
		case strings.HasPrefix(part, ValidationMin+":"):
			limit, err := parseValidationLimit(part, ValidationMin)
			if err != nil {
				return nil, err
			}
			rules = append(rules, validationRule{kind: ValidationMin, limit: limit})
		case strings.HasPrefix(part, ValidationMax+":"):
			limit, err := parseValidationLimit(part, ValidationMax)
			if err != nil {
				return nil, err
			}
			rules = append(rules, validationRule{kind: ValidationMax, limit: limit})
		default:
			return nil, fmt.Errorf("lazyerrors: unsupported validation rule %q", part)
		}
	}
	return rules, nil
}

func parseValidationLimit(part string, kind string) (int, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(part, kind+":"))
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("lazyerrors: invalid %s validation limit %q", kind, raw)
	}
	if limit < 0 {
		return 0, fmt.Errorf("lazyerrors: %s validation limit must be non-negative", kind)
	}
	return limit, nil
}

func validationFieldName(root any, fieldPath []string) string {
	name, err := lazyschema.FieldNameFor(root, strings.Join(fieldPath, "."))
	if err == nil {
		return name
	}
	return strings.Join(fieldPath, "_")
}

func validationPresent(value reflect.Value) bool {
	value = indirectValue(value)
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.String:
		return strings.TrimSpace(value.String()) != ""
	case reflect.Array, reflect.Map, reflect.Slice:
		return value.Len() > 0
	default:
		return !value.IsZero()
	}
}

func validationSize(value reflect.Value) (int, bool) {
	value = indirectValue(value)
	if !value.IsValid() {
		return 0, true
	}
	switch value.Kind() {
	case reflect.String:
		return utf8.RuneCountInString(value.String()), true
	case reflect.Array, reflect.Map, reflect.Slice:
		return value.Len(), true
	default:
		return 0, false
	}
}

func shouldRecurseValidation(value reflect.Value) bool {
	value = indirectValue(value)
	if !value.IsValid() {
		return false
	}
	t := value.Type()
	return t.Kind() == reflect.Struct && t != timeType
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func collectValidationErrors(err error, out *[]ValidationError) {
	if err == nil {
		return
	}
	switch validation := err.(type) {
	case ValidationError:
		*out = append(*out, validation)
		return
	case *ValidationError:
		if validation != nil {
			*out = append(*out, *validation)
		}
		return
	}
	if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		for _, wrapped := range unwrapper.Unwrap() {
			collectValidationErrors(wrapped, out)
		}
		return
	}
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		collectValidationErrors(unwrapper.Unwrap(), out)
	}
}

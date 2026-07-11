package lazyconfig

import (
	"encoding"
	"fmt"
	"maps"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
var durationType = reflect.TypeFor[time.Duration]()

type validator interface {
	Validate() error
}

type loader struct {
	env map[string]string
}

// Option changes how Getenv and MustGetenv read environment variables.
type Option func(*loader)

// Getenv fills a configuration struct from environment variables.
func Getenv[T any](options ...Option) (T, error) {
	var config T
	value := reflect.ValueOf(&config).Elem()
	loader := newLoader(options...)
	if err := loader.fillRoot(value); err != nil {
		return config, err
	}
	if validatable, ok := any(&config).(validator); ok {
		return config, validatable.Validate()
	}
	if validatable, ok := any(config).(validator); ok {
		return config, validatable.Validate()
	}
	return config, nil
}

// MustGetenv fills a configuration struct from environment variables and
// panics when the environment is invalid.
func MustGetenv[T any](options ...Option) T {
	config, err := Getenv[T](options...)
	if err != nil {
		panic(err)
	}
	return config
}

// RemoveEnvNamePrefix makes environment variables with prefix available without
// that prefix. RemoveEnvNamePrefix("OTEL") lets field SDKDisabled match
// OTEL_SDK_DISABLED as SDK_DISABLED.
//
// Existing unprefixed environment variables keep precedence over generated
// aliases.
func RemoveEnvNamePrefix(prefix string) Option {
	prefix = strings.TrimSpace(strings.TrimSuffix(prefix, "_"))
	return func(loader *loader) {
		if prefix == "" {
			return
		}
		marker := prefix + "_"
		aliases := make(map[string]string)
		for name, value := range loader.env {
			if !strings.HasPrefix(name, marker) {
				continue
			}
			alias := strings.TrimPrefix(name, marker)
			if alias == "" {
				continue
			}
			if _, ok := loader.env[alias]; ok {
				continue
			}
			aliases[alias] = value
		}
		maps.Copy(loader.env, aliases)
	}
}

func newLoader(options ...Option) loader {
	env := make(map[string]string)
	for _, item := range os.Environ() {
		name, value, ok := strings.Cut(item, "=")
		if ok {
			env[name] = value
		}
	}
	loader := loader{env: env}
	for _, option := range options {
		if option != nil {
			option(&loader)
		}
	}
	return loader
}

func (l loader) fillRoot(value reflect.Value) error {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("lazyconfig: config type must be a struct or pointer to struct")
	}
	return l.fillStruct(value, "")
}

func (l loader) fillStruct(value reflect.Value, prefix string) error {
	typ := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := typ.Field(index)
		if !field.IsExported() {
			continue
		}
		if err := l.fillField(value.Field(index), field, prefix); err != nil {
			return err
		}
	}
	return nil
}

func (l loader) fillField(value reflect.Value, field reflect.StructField, prefix string) error {
	if !value.CanSet() {
		return nil
	}

	if value.Kind() == reflect.Pointer {
		return l.fillPointer(value, field, prefix)
	}

	if canSetScalar(value) {
		return l.fillScalar(value, fieldEnvNames(prefix, field), field)
	}

	switch value.Kind() {
	case reflect.Struct:
		return l.fillStruct(value, fieldEnvName(prefix, field))
	case reflect.Slice:
		return l.fillSlice(value, sliceEnvName(prefix, field), field)
	default:
		return fmt.Errorf("lazyconfig: %s has unsupported type %s", field.Name, field.Type)
	}
}

func (l loader) fillPointer(value reflect.Value, field reflect.StructField, prefix string) error {
	elemType := value.Type().Elem()
	if elemType.Kind() == reflect.Struct && !canSetScalarType(elemType) {
		envName := fieldEnvName(prefix, field)
		if !l.hasStructEnv(envName, elemType) {
			if required, reason := requirement(field); required {
				return requiredError(envName, reason)
			}
			return nil
		}
		value.Set(reflect.New(elemType))
		return l.fillStruct(value.Elem(), envName)
	}

	envName, raw, found := l.lookupValue(fieldEnvNames(prefix, field), field)
	if !found {
		if required, reason := requirement(field); required {
			return requiredError(envName, reason)
		}
		return nil
	}
	value.Set(reflect.New(elemType))
	return setScalar(value.Elem(), envName, raw)
}

func (l loader) fillScalar(value reflect.Value, envNames []string, field reflect.StructField) error {
	envName, raw, found := l.lookupValue(envNames, field)
	required, reason := requirement(field)
	if required && (!found || strings.TrimSpace(raw) == "") {
		return requiredError(envName, reason)
	}
	if !found {
		return nil
	}
	return setScalar(value, envName, raw)
}

func (l loader) fillSlice(value reflect.Value, prefix string, field reflect.StructField) error {
	elemType := value.Type().Elem()
	var values []reflect.Value

	if elemType.Kind() == reflect.Struct && !canSetScalarType(elemType) {
		if l.hasDirectStructEnv(prefix, elemType) {
			item := reflect.New(elemType).Elem()
			if err := l.fillStruct(item, prefix); err != nil {
				return err
			}
			values = append(values, item)
		}
		for _, index := range l.structIndexes(prefix) {
			item := reflect.New(elemType).Elem()
			if err := l.fillStruct(item, fmt.Sprintf("%s_%d", prefix, index)); err != nil {
				return err
			}
			values = append(values, item)
		}
	} else if canSetScalarType(elemType) {
		if envName, raw, ok := l.lookupValue([]string{prefix}, field); ok {
			if elemType.Kind() == reflect.String {
				for _, part := range splitStringSlice(raw) {
					item := reflect.New(elemType).Elem()
					item.SetString(part)
					values = append(values, item)
				}
			} else {
				item := reflect.New(elemType).Elem()
				if err := setScalar(item, envName, raw); err != nil {
					return err
				}
				values = append(values, item)
			}
		}
		for _, index := range l.scalarIndexes(prefix) {
			envName := fmt.Sprintf("%s_%d", prefix, index)
			raw, ok := l.env[envName]
			if !ok {
				continue
			}
			item := reflect.New(elemType).Elem()
			if err := setScalar(item, envName, raw); err != nil {
				return err
			}
			values = append(values, item)
		}
	} else {
		return fmt.Errorf("lazyconfig: %s has unsupported slice element type %s", field.Name, elemType)
	}

	if len(values) == 0 {
		if required, reason := requirement(field); required {
			return requiredError(prefix, reason)
		}
		return nil
	}

	result := reflect.MakeSlice(value.Type(), 0, len(values))
	for _, item := range values {
		result = reflect.Append(result, item)
	}
	value.Set(result)
	return nil
}

func (l loader) lookupValue(envNames []string, field reflect.StructField) (string, string, bool) {
	for _, envName := range envNames {
		if value, ok := l.env[envName]; ok {
			return envName, strings.TrimSpace(value), true
		}
	}
	if value, ok := field.Tag.Lookup("default"); ok {
		return envNames[0], strings.TrimSpace(value), true
	}
	return envNames[0], "", false
}

func (l loader) hasDirectStructEnv(prefix string, typ reflect.Type) bool {
	for field := range typ.Fields() {
		if !field.IsExported() {
			continue
		}
		for _, envName := range fieldEnvNames(prefix, field) {
			if _, ok := l.env[envName]; ok {
				return true
			}
			if field.Type.Kind() == reflect.Struct && !canSetScalarType(field.Type) {
				if l.hasDirectStructEnv(envName, field.Type) {
					return true
				}
			}
		}
	}
	return false
}

func (l loader) hasStructEnv(prefix string, typ reflect.Type) bool {
	if l.hasDirectStructEnv(prefix, typ) {
		return true
	}
	marker := prefix + "_"
	for name := range l.env {
		if strings.HasPrefix(name, marker) {
			return true
		}
	}
	return false
}

func (l loader) structIndexes(prefix string) []int {
	marker := prefix + "_"
	seen := make(map[int]bool)
	for name := range l.env {
		if !strings.HasPrefix(name, marker) {
			continue
		}
		rest := name[len(marker):]
		digitCount := 0
		for _, char := range rest {
			if char < '0' || char > '9' {
				break
			}
			digitCount++
		}
		if digitCount == 0 || len(rest) == digitCount || rest[digitCount] != '_' {
			continue
		}
		index, err := strconv.Atoi(rest[:digitCount])
		if err == nil {
			seen[index] = true
		}
	}
	out := make([]int, 0, len(seen))
	for index := range seen {
		out = append(out, index)
	}
	sort.Ints(out)
	return out
}

func (l loader) scalarIndexes(prefix string) []int {
	marker := prefix + "_"
	seen := make(map[int]bool)
	for name := range l.env {
		if !strings.HasPrefix(name, marker) {
			continue
		}
		rest := name[len(marker):]
		if rest == "" {
			continue
		}
		index, err := strconv.Atoi(rest)
		if err == nil {
			seen[index] = true
		}
	}
	out := make([]int, 0, len(seen))
	for index := range seen {
		out = append(out, index)
	}
	sort.Ints(out)
	return out
}

func canSetScalar(value reflect.Value) bool {
	return canSetScalarType(value.Type())
}

func canSetScalarType(typ reflect.Type) bool {
	if typ == durationType {
		return true
	}
	if reflect.PointerTo(typ).Implements(textUnmarshalerType) || typ.Implements(textUnmarshalerType) {
		return true
	}
	switch typ.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func setScalar(value reflect.Value, envName string, raw string) error {
	raw = strings.TrimSpace(raw)
	if value.Type() == durationType {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("lazyconfig: parse %s: %w", envName, err)
		}
		value.SetInt(int64(parsed))
		return nil
	}

	if value.CanAddr() && reflect.PointerTo(value.Type()).Implements(textUnmarshalerType) {
		unmarshaler := value.Addr().Interface().(encoding.TextUnmarshaler)
		if err := unmarshaler.UnmarshalText([]byte(raw)); err != nil {
			return fmt.Errorf("lazyconfig: parse %s: %w", envName, err)
		}
		return nil
	}
	if value.Type().Implements(textUnmarshalerType) {
		unmarshaler := value.Interface().(encoding.TextUnmarshaler)
		if err := unmarshaler.UnmarshalText([]byte(raw)); err != nil {
			return fmt.Errorf("lazyconfig: parse %s: %w", envName, err)
		}
		return nil
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(raw)
	case reflect.Bool:
		value.SetBool(parseBool(raw))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("lazyconfig: parse %s: %w", envName, err)
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(raw, 10, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("lazyconfig: parse %s: %w", envName, err)
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("lazyconfig: parse %s: %w", envName, err)
		}
		value.SetFloat(parsed)
	default:
		return fmt.Errorf("lazyconfig: %s has unsupported type %s", envName, value.Type())
	}
	return nil
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "yes", "true", "1":
		return true
	case "no", "false", "0":
		return false
	default:
		return false
	}
}

func splitStringSlice(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return strings.FieldsFunc(raw, func(char rune) bool {
		return char == ',' || unicode.IsSpace(char)
	})
}

func fieldEnvName(prefix string, field reflect.StructField) string {
	return fieldEnvNames(prefix, field)[0]
}

func fieldEnvNames(prefix string, field reflect.StructField) []string {
	names := envNames(field)
	if prefix == "" || hasVarTag(field) {
		return names
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, prefix+"_"+name)
	}
	return out
}

func sliceEnvName(prefix string, field reflect.StructField) string {
	if hasVarTag(field) {
		return envName(field)
	}
	name := singularEnvName(envName(field))
	if prefix == "" {
		return name
	}
	return prefix + "_" + name
}

func envName(field reflect.StructField) string {
	return envNames(field)[0]
}

func envNames(field reflect.StructField) []string {
	if value, ok := field.Tag.Lookup("var"); ok && strings.TrimSpace(value) != "" {
		return []string{strings.TrimSpace(value)}
	}
	name := camelEnvName(field.Name)
	compact := strings.ReplaceAll(name, "_", "")
	if compact != name {
		return []string{name, compact}
	}
	return []string{name}
}

func hasVarTag(field reflect.StructField) bool {
	value, ok := field.Tag.Lookup("var")
	return ok && strings.TrimSpace(value) != ""
}

func camelEnvName(name string) string {
	var out []rune
	runes := []rune(name)
	for index, char := range runes {
		if index > 0 && shouldSplit(runes, index) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToUpper(char))
	}
	return string(out)
}

func shouldSplit(runes []rune, index int) bool {
	current := runes[index]
	previous := runes[index-1]
	if !unicode.IsUpper(current) && !unicode.IsDigit(current) {
		return false
	}
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}
	return unicode.IsUpper(previous) && index+1 < len(runes) && unicode.IsLower(runes[index+1])
}

func singularEnvName(name string) string {
	if strings.HasSuffix(name, "IES") && len(name) > 3 {
		return strings.TrimSuffix(name, "IES") + "Y"
	}
	if strings.HasSuffix(name, "S") && len(name) > 1 {
		return strings.TrimSuffix(name, "S")
	}
	return name
}

func requirement(field reflect.StructField) (bool, string) {
	for _, key := range []string{"required", "require"} {
		value, ok := field.Tag.Lookup(key)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" || strings.EqualFold(value, "true") {
			return true, ""
		}
		if strings.EqualFold(value, "false") {
			continue
		}
		return true, value
	}
	return false, ""
}

func requiredError(envName string, reason string) error {
	if reason == "" {
		return fmt.Errorf("%s missing", envName)
	}
	if strings.HasPrefix(strings.ToLower(reason), "for ") {
		return fmt.Errorf("%s is required %s, please set", envName, reason)
	}
	return fmt.Errorf("%s is required for %s, please set", envName, reason)
}

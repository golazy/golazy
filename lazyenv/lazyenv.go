// Package lazyenv provides easy access to the environment
package lazyenv

import (
	"os"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const (
	production  string = "production"
	development string = "development"
)

// Env returns Development if the environment is a terminal or the ENVIRONMENT variable starts with "dev".
// If the environment variable does not starts with dev, it will assume it is development if the output is a terminal.
// Otherwise it will return Production.
// Env can only return the strings "development" or "production". If you want to access the environment directly, use os.Getenv("ENVIRONMENT")
func Env() string {
	isDev := strings.HasPrefix(os.Getenv("ENVIRONMENT"), "dev")
	if isDev {
		return development
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return development
	}
	return production

}

// Get returns the value of an environment variable and tries to convert it to the type of the argument.
func Get[T any](key string) T {
	v := new(T)
	setFieldValue(reflect.ValueOf(v).Elem(), os.Getenv(key))
	return *v
}

// GetDefault returns the value of an environment variable and tries to convert it to the type of the argument.
func GetDefault[T any](key string, defaultValue T) T {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return Get[T](key)
}

// IsProduction returns true if the environment is production
func IsProduction() bool {
	return Env() == production

}

// IsDevelopment returns true if the environment is development
func IsDevelopment() bool {
	return Env() == development
}

// Fill fills the fields of the struct with the values from the environment.
// It will use the uppercase and dash separated name of the field as the environment variable name.
// For example, if the struct has a field named "DBName", it will look for the environment variable "DB_NAME".
// It will try to convert the value to the type of the field.
// If the field is a pointer, it will try to convert the value to the type of the pointer.
// If the field is a slice, it will split the value by commas.
// If the field is a map, it will split the value by commas and then by colons.
// If the field is a struct, it will recursively fill the fields of the struct using the field name as a prefix.
// The name of the environment variable can be overridden by the "env" tag in the struct field.
func Fill(dest interface{}) {
	fillWithPrefix(dest, "")
}

func fillWithPrefix(dest interface{}, prefix string) {
	v := reflect.ValueOf(dest).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		envName := field.Tag.Get("env")
		if envName == "" {
			envName = toEnvName(field.Name)
		}
		envKey := prefix + envName

		envValue := os.Getenv(envKey)
		if envValue != "" {
			setFieldValue(fieldValue, envValue)
		}

		// Check if the field is a struct or a pointer to a struct
		if fieldValue.Kind() == reflect.Struct || (fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct) {
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
				}
				fieldValue = fieldValue.Elem()
			}
			fillWithPrefix(fieldValue.Addr().Interface(), prefix+envName+"_")
		}
	}
}

// toEnvName converts a field name to an environment variable name.
func toEnvName(name string) string {
	var result string
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			if name[i-1] >= 'a' && name[i-1] <= 'z' || (i+1 < len(name) && name[i+1] >= 'a' && name[i+1] <= 'z') {
				result += "_"
			}
		}
		result += string(r)
	}
	return strings.ToUpper(result)
}

// setFieldValue sets the value of a field based on the environment variable value.
func setFieldValue(fieldValue reflect.Value, envValue string) {
	switch fieldValue.Kind() {
	case reflect.Ptr:
		ptrValue := reflect.New(fieldValue.Type().Elem())
		setFieldValue(ptrValue.Elem(), envValue)
		fieldValue.Set(ptrValue)
	case reflect.Slice:
		values := strings.Split(envValue, ",")
		slice := reflect.MakeSlice(fieldValue.Type(), len(values), len(values))
		for i, value := range values {
			setFieldValue(slice.Index(i), value)
		}
		fieldValue.Set(slice)
	case reflect.Map:
		elemType := fieldValue.Type().Elem()
		keyType := fieldValue.Type().Key()
		mapValue := reflect.MakeMap(fieldValue.Type())
		pairs := strings.Split(envValue, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := reflect.New(keyType).Elem()
			setFieldValue(key, kv[0])
			value := reflect.New(elemType).Elem()
			setFieldValue(value, kv[1])
			mapValue.SetMapIndex(key, value)
		}
		fieldValue.Set(mapValue)
	case reflect.String:
		fieldValue.SetString(envValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intValue, err := strconv.ParseInt(envValue, 10, 64); err == nil {
			fieldValue.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintValue, err := strconv.ParseUint(envValue, 10, 64); err == nil {
			fieldValue.SetUint(uintValue)
		}
	case reflect.Float32, reflect.Float64:
		if floatValue, err := strconv.ParseFloat(envValue, 64); err == nil {
			fieldValue.SetFloat(floatValue)
		}
	case reflect.Bool:
		if boolValue, err := strconv.ParseBool(envValue); err == nil {
			fieldValue.SetBool(boolValue)
		}
	}
}

// Port returns a valid Addr to listen to the service
// It defaults to "127.0.0.1:2000"
// If the PORT environment variable is set, it will use it as the port listening in all adress
func Addr() string {
	p := os.Getenv("PORT")
	if p == "" {
		return "127.0.0.1:2000"
	}
	// PORT can be a number or a colon separated address
	if _, err := strconv.Atoi(p); err == nil {
		return ":" + p
	}
	panic("invalid port")
}

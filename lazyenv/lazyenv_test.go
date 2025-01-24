package lazyenv

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestFillWithNestedStructs(t *testing.T) {
	envVars := map[string]string{
		"DB_NAME": "test_db",
		"DB_PASS": "secret",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	type DBConfig struct {
		Name     string
		Password string `env:"PASS"`
	}

	type Config struct {
		DB  *DBConfig
		DB2 DBConfig `env:"DB"`
	}

	expect := Config{
		DB: &DBConfig{
			Name:     "test_db",
			Password: "secret",
		},
		DB2: DBConfig{
			Name:     "test_db",
			Password: "secret",
		},
	}

	var config Config
	Fill(&config)

	if config.DB == nil || config.DB.Name != expect.DB.Name || config.DB.Password != expect.DB.Password {
		t.Errorf("Fill() DB = %+v; expected %+v", config.DB, expect.DB)
	}

	if config.DB2.Name != expect.DB2.Name || config.DB2.Password != expect.DB2.Password {
		t.Errorf("Fill() DB2 = %+v; expected %+v", config.DB2, expect.DB2)
	}
	for key := range envVars {
		os.Unsetenv(key)
	}
}

func TestToEnvName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"DBName", "DB_NAME"},
		{"UserID", "USER_ID"},
		{"HTTPServer", "HTTP_SERVER"},
		{"MyVar", "MY_VAR"},
		{"simple", "SIMPLE"},
		{"AnotherExample", "ANOTHER_EXAMPLE"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := toEnvName(test.input)
			if result != test.expected {
				t.Errorf("toEnvName(%s) = %s; expected %s", test.input, result, test.expected)
			}
		})
	}
}

func TestFillFromEnvWithTags(t *testing.T) {
	envVars := map[string]string{
		"DB_NAME":     "test_db",
		"USER_ID":     "42",
		"HTTP_SERVER": "localhost",
		"MY_VAR":      "some_value",
		"SIMPLE":      "true",
		"ANOTHER":     "1,2,3",
		"MAP_EXAMPLE": "key1:value1,key2:value2",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	type Config struct {
		EnvDBName     string            `env:"DB_NAME"`
		EnvUserID     int               `env:"USER_ID"`
		EnvHTTPServer string            `env:"HTTP_SERVER"`
		EnvMyVar      *string           `env:"MY_VAR"`
		EnvSimple     bool              `env:"SIMPLE"`
		EnvAnother    []int             `env:"ANOTHER"`
		EnvMapExample map[string]string `env:"MAP_EXAMPLE"`
	}

	expect := Config{
		EnvDBName:     "test_db",
		EnvUserID:     42,
		EnvHTTPServer: "localhost",
		EnvMyVar:      strPtr("some_value"),
		EnvSimple:     true,
		EnvAnother:    []int{1, 2, 3},
		EnvMapExample: map[string]string{"key1": "value1", "key2": "value2"},
	}

	var config Config
	Fill(&config)

	if !reflect.DeepEqual(config, expect) {
		t.Errorf("FillFromEnv() = %+v; expected %+v", config, expect)
	}

	for key := range envVars {
		os.Unsetenv(key)
	}
}

func TestFillFromEnvWithoutTags(t *testing.T) {
	envVars := map[string]string{
		"DB_NAME":     "test_db",
		"USER_ID":     "42",
		"HTTP_SERVER": "localhost",
		"MY_VAR":      "some_value",
		"SIMPLE":      "true",
		"ANOTHER":     "1,2,3",
		"MAP_EXAMPLE": "key1:value1,key2:value2",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	type Config struct {
		DBName     string
		UserID     int
		HTTPServer string
		MyVar      *string
		Simple     bool
		Another    []int
		MapExample map[string]string
	}

	expect := Config{
		DBName:     "test_db",
		UserID:     42,
		HTTPServer: "localhost",
		MyVar:      strPtr("some_value"),
		Simple:     true,
		Another:    []int{1, 2, 3},
		MapExample: map[string]string{"key1": "value1", "key2": "value2"},
	}

	var config Config
	Fill(&config)

	if !reflect.DeepEqual(config, expect) {
		t.Errorf("FillFromEnv() = %+v; expected %+v", config, expect)
	}

	for key := range envVars {
		os.Unsetenv(key)
	}
}

func strPtr(s string) *string {
	return &s
}

func ExampleFill() {
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PASS", "localhost")
	os.Setenv("USER_ID", "42")
	os.Setenv("HTTP_SERVER", "localhost")
	os.Setenv("SIMPLE", "true")
	os.Setenv("ANOTHER", "1,2,3")
	os.Setenv("MAP_EXAMPLE", "key1:value1,key2:value2")
	os.Setenv("lowercase", "with_env")

	type Config struct {
		DB struct {
			Name     string
			Host     string
			Password string `env:"PASS"`
		}
		UserID     int
		HTTPServer string
		Simple     bool
		Another    []int
		MapExample map[string]string
		LowerCase  string `env:"lowercase"`
	}

	var config Config
	Fill(&config)

	fmt.Printf("%+v\n", config)
	// Output:
	//{DB:{Name:test_db Host:localhost Password:localhost} UserID:42 HTTPServer:localhost Simple:true Another:[1 2 3] MapExample:map[key1:value1 key2:value2] LowerCase:with_env}
}

func TestGet(t *testing.T) {
	os.Setenv("DB_NAME", "3")

	i := Get[int]("DB_NAME")
	if i != 3 {
		t.Errorf("Get() = %d; expected 3", i)
	}
	i = Get[int]("MISSING")
	if i != 0 {
		t.Errorf("Get() = %d; expected 3", i)
	}

}

func TestGetDefault(t *testing.T) {
	os.Setenv("DB_NAME", "3")

	i := Get[int]("DB_NAME")
	if i != 3 {
		t.Errorf("Get() = %d; expected 3", i)
	}
	i = GetDefault("DB_NAME", 5)
	if i != 3 {
		t.Errorf("Get() = %d; expected 3", i)
	}
	i = GetDefault("MISSING", 3)
	if i != 3 {
		t.Errorf("Get() = %d; expected 3", i)
	}

}

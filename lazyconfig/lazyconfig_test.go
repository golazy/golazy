package lazyconfig

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGetenvFillsScalarFields(t *testing.T) {
	t.Setenv("PORT", "1234")
	t.Setenv("LISTEN_ADDR", "127.0.0.1:1234")
	t.Setenv("DEBUG", "true")
	t.Setenv("TIMEOUT", "5s")

	type config struct {
		Port       int
		ListenAddr string
		Debug      bool
		Timeout    time.Duration
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	want := config{
		Port:       1234,
		ListenAddr: "127.0.0.1:1234",
		Debug:      true,
		Timeout:    5 * time.Second,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config = %#v, want %#v", got, want)
	}
}

func TestGetenvUsesTags(t *testing.T) {
	t.Setenv("LAZYCONFIG_TEST_HOST", "localhost")

	type config struct {
		Host string `var:"LAZYCONFIG_TEST_HOST"`
		Mode string `default:"development"`
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "localhost" || got.Mode != "development" {
		t.Fatalf("config = %#v", got)
	}
}

func TestGetenvUsesCompactDefaultNameFallback(t *testing.T) {
	t.Setenv("GOWORK", "/tmp/app/go.work")
	t.Setenv("LAZYCMD", "/tmp/lazy")

	type config struct {
		GoWork  string
		LazyCmd string
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	if got.GoWork != "/tmp/app/go.work" {
		t.Fatalf("GoWork = %q", got.GoWork)
	}
	if got.LazyCmd != "/tmp/lazy" {
		t.Fatalf("LazyCmd = %q", got.LazyCmd)
	}
}

func TestGetenvFillsIndexedSliceSortedByIndex(t *testing.T) {
	t.Setenv("LAZYCONFIG_TEST_LISTENER_20_NAME", "smtp")
	t.Setenv("LAZYCONFIG_TEST_LISTENER_20_PORT", "25")
	t.Setenv("LAZYCONFIG_TEST_LISTENER_10_NAME", "http")
	t.Setenv("LAZYCONFIG_TEST_LISTENER_10_PORT", "123")

	type listener struct {
		Name string
		Port int
	}
	type config struct {
		LazyconfigTestListeners []listener
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	want := []listener{
		{Name: "http", Port: 123},
		{Name: "smtp", Port: 25},
	}
	if !reflect.DeepEqual(got.LazyconfigTestListeners, want) {
		t.Fatalf("listeners = %#v, want %#v", got.LazyconfigTestListeners, want)
	}
}

func TestGetenvFillsSingleSliceItem(t *testing.T) {
	t.Setenv("LAZYCONFIG_SINGLE_LISTENER_NAME", "http")
	t.Setenv("LAZYCONFIG_SINGLE_LISTENER_PORT", "123")

	type listener struct {
		Name string
		Port int
	}
	type config struct {
		Listeners []listener `var:"LAZYCONFIG_SINGLE_LISTENER"`
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	want := []listener{{Name: "http", Port: 123}}
	if !reflect.DeepEqual(got.Listeners, want) {
		t.Fatalf("listeners = %#v, want %#v", got.Listeners, want)
	}
}

func TestGetenvFillsScalarSliceItems(t *testing.T) {
	t.Setenv("LAZYCONFIG_SCALAR_NAME_2", "second")
	t.Setenv("LAZYCONFIG_SCALAR_NAME_1", "first")

	type config struct {
		Names []string `var:"LAZYCONFIG_SCALAR_NAME"`
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"first", "second"}
	if !reflect.DeepEqual(got.Names, want) {
		t.Fatalf("names = %#v, want %#v", got.Names, want)
	}
}

func TestGetenvRequiredErrors(t *testing.T) {
	type missingConfig struct {
		Value string `var:"LAZYCONFIG_REQUIRED_MISSING" required:"true"`
	}
	_, err := Getenv[missingConfig]()
	if err == nil || err.Error() != "LAZYCONFIG_REQUIRED_MISSING missing" {
		t.Fatalf("error = %v", err)
	}

	type reasonConfig struct {
		Value string `var:"LAZYCONFIG_REQUIRED_REASON" require:"for dancing"`
	}
	_, err = Getenv[reasonConfig]()
	if err == nil || err.Error() != "LAZYCONFIG_REQUIRED_REASON is required for dancing, please set" {
		t.Fatalf("error = %v", err)
	}
}

func TestGetenvCallsValidate(t *testing.T) {
	t.Setenv("LAZYCONFIG_VALIDATE_PORT", "0")

	_, err := Getenv[validatingConfig]()
	if err == nil || !strings.Contains(err.Error(), "port must be positive") {
		t.Fatalf("error = %v", err)
	}
}

type validatingConfig struct {
	Port int `var:"LAZYCONFIG_VALIDATE_PORT"`
}

func (c validatingConfig) Validate() error {
	if c.Port <= 0 {
		return errors.New("port must be positive")
	}
	return nil
}

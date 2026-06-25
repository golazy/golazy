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

func TestGetenvTrimsValuesAndFillsPointerScalars(t *testing.T) {
	t.Setenv("LAZYCONFIG_TRIM_NAME", "  api  ")
	t.Setenv("LAZYCONFIG_TRIM_PORT", "  8080  ")
	t.Setenv("LAZYCONFIG_TRIM_LIMIT", "  12  ")
	t.Setenv("LAZYCONFIG_TRIM_ENABLED", "  true  ")

	type config struct {
		Name    string `var:"LAZYCONFIG_TRIM_NAME"`
		Port    int    `var:"LAZYCONFIG_TRIM_PORT"`
		Limit   *uint  `var:"LAZYCONFIG_TRIM_LIMIT"`
		Enabled *bool  `var:"LAZYCONFIG_TRIM_ENABLED"`
		Missing *int   `var:"LAZYCONFIG_TRIM_MISSING"`
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "api" {
		t.Fatalf("Name = %q", got.Name)
	}
	if got.Port != 8080 {
		t.Fatalf("Port = %d", got.Port)
	}
	if got.Limit == nil || *got.Limit != 12 {
		t.Fatalf("Limit = %#v", got.Limit)
	}
	if got.Enabled == nil || !*got.Enabled {
		t.Fatalf("Enabled = %#v", got.Enabled)
	}
	if got.Missing != nil {
		t.Fatalf("Missing = %#v, want nil", got.Missing)
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

func TestGetenvRemovesEnvNamePrefix(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "sample")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://collector:4318")

	type otlpConfig struct {
		TracesEndpoint string
	}
	type exporterConfig struct {
		OTLP otlpConfig
	}
	type config struct {
		ServiceName string
		Exporter    exporterConfig
	}

	got := MustGetenv[config](RemoveEnvNamePrefix("OTEL"))
	if got.ServiceName != "sample" {
		t.Fatalf("ServiceName = %q, want sample", got.ServiceName)
	}
	if got.Exporter.OTLP.TracesEndpoint != "http://collector:4318" {
		t.Fatalf("TracesEndpoint = %q", got.Exporter.OTLP.TracesEndpoint)
	}
}

func TestRemoveEnvNamePrefixKeepsUnprefixedPrecedence(t *testing.T) {
	t.Setenv("SERVICE_NAME", "direct")
	t.Setenv("OTEL_SERVICE_NAME", "otel")

	type config struct {
		ServiceName string
	}

	got := MustGetenv[config](RemoveEnvNamePrefix("OTEL_"))
	if got.ServiceName != "direct" {
		t.Fatalf("ServiceName = %q, want direct", got.ServiceName)
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

func TestGetenvFillsStringSliceFromDelimitedValue(t *testing.T) {
	t.Setenv("LAZYCONFIG_SLICE_VALUES", " alpha,beta  gamma , delta ")

	type config struct {
		Values []string `var:"LAZYCONFIG_SLICE_VALUES"`
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "beta", "gamma", "delta"}
	if !reflect.DeepEqual(got.Values, want) {
		t.Fatalf("Values = %#v, want %#v", got.Values, want)
	}
}

func TestGetenvBoolValues(t *testing.T) {
	t.Setenv("LAZYCONFIG_BOOL_YES", "YeS")
	t.Setenv("LAZYCONFIG_BOOL_TRUE", "TrUe")
	t.Setenv("LAZYCONFIG_BOOL_ONE", "1")
	t.Setenv("LAZYCONFIG_BOOL_NO", "nO")
	t.Setenv("LAZYCONFIG_BOOL_FALSE", "FaLsE")
	t.Setenv("LAZYCONFIG_BOOL_ZERO", "0")
	t.Setenv("LAZYCONFIG_BOOL_UNKNOWN", "on")
	t.Setenv("LAZYCONFIG_BOOL_POINTER_UNKNOWN", "maybe")

	type config struct {
		Yes            bool  `var:"LAZYCONFIG_BOOL_YES"`
		True           bool  `var:"LAZYCONFIG_BOOL_TRUE"`
		One            bool  `var:"LAZYCONFIG_BOOL_ONE"`
		No             bool  `var:"LAZYCONFIG_BOOL_NO"`
		False          bool  `var:"LAZYCONFIG_BOOL_FALSE"`
		Zero           bool  `var:"LAZYCONFIG_BOOL_ZERO"`
		Unknown        bool  `var:"LAZYCONFIG_BOOL_UNKNOWN"`
		PointerUnknown *bool `var:"LAZYCONFIG_BOOL_POINTER_UNKNOWN"`
		PointerMissing *bool `var:"LAZYCONFIG_BOOL_POINTER_MISSING"`
	}

	got, err := Getenv[config]()
	if err != nil {
		t.Fatal(err)
	}
	if !got.Yes {
		t.Fatalf("Yes = false")
	}
	if !got.True {
		t.Fatalf("True = false")
	}
	if !got.One {
		t.Fatalf("One = false")
	}
	if got.No {
		t.Fatalf("No = true")
	}
	if got.False {
		t.Fatalf("False = true")
	}
	if got.Zero {
		t.Fatalf("Zero = true")
	}
	if got.Unknown {
		t.Fatalf("Unknown = true")
	}
	if got.PointerUnknown == nil || *got.PointerUnknown {
		t.Fatalf("PointerUnknown = %#v", got.PointerUnknown)
	}
	if got.PointerMissing != nil {
		t.Fatalf("PointerMissing = %#v, want nil", got.PointerMissing)
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

func TestMustGetenvPanicsForInvalidConfig(t *testing.T) {
	t.Setenv("LAZYCONFIG_MUST_PORT", "abc")

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("MustGetenv did not panic")
		}
		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("panic = %#v, want error", recovered)
		}
		if !strings.Contains(err.Error(), "parse LAZYCONFIG_MUST_PORT") {
			t.Fatalf("panic = %v", recovered)
		}
	}()

	_ = MustGetenv[struct {
		Port int `var:"LAZYCONFIG_MUST_PORT"`
	}]()
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

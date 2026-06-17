package lazyschema

import (
	"reflect"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

func registerDefaultConverters(decoder *Decoder) {
	decoder.RegisterConverter(time.Time{}, convertTime)
}

func registerDefaultEncoders(encoder *Encoder) {
	encoder.RegisterEncoder(time.Time{}, func(v reflect.Value) string {
		t, ok := v.Interface().(time.Time)
		if !ok || t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339)
	})
}

func convertTime(value string) reflect.Value {
	if value == "" {
		return reflect.Zero(timeType)
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
		"15:04:05",
		"15:04",
		"2006-01",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return reflect.ValueOf(parsed)
		}
	}
	return invalidValue
}

// FormatTime formats a time value for HTML date and time inputs.
func FormatTime(t time.Time, inputType string) string {
	if t.IsZero() {
		return ""
	}
	switch inputType {
	case "date":
		return t.Format("2006-01-02")
	case "time":
		return t.Format("15:04")
	case "datetime", "datetime-local":
		return t.Format("2006-01-02T15:04")
	case "month":
		return t.Format("2006-01")
	default:
		return t.Format(time.RFC3339)
	}
}

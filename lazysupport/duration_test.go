package lazysupport

import (
	"testing"
	"time"
)

var DurationTest = []struct {
	Value  time.Duration
	Output string
}{
	{0, "0s"},
	{1, "1ns"},
	{12, "12ns"},
	{123, "123ns"},
	{1000, "1µs"},
	{1200, "1.2µs"},
	{1234, "1.23µs"},
	{10000, "10µs"},
	{10123, "10.1µs"},
	{123456, "123µs"},
	{1000000, "1ms"},
	{1234567, "1.23ms"},
	{12345678, "12.3ms"},
	{123456789, "123ms"},
	{1234567890, "1.23s"},
	{12345678901, "12.3s"},
	{12345678901, "12.3s"},
	{time.Minute, "1m"},
	{time.Minute + time.Second, "1m 1s"},
	{time.Hour + time.Minute + time.Second, "1h 1m"},
	{day + time.Hour, "1d 1h"},
	{Month, "1M"},
	{Month + day, "1M 1d"},
	{Year + day, "1y"},
	{Year + Month + day, "1y 1M"},
}

func TestDuration(t *testing.T) {
	for _, test := range DurationTest {
		t.Run(test.Output, func(t *testing.T) {

			result := ShortDuration(test.Value)
			if result != test.Output {
				t.Error("❗", uint64(test.Value), result, "!=", test.Output)
			} else {
				t.Log("✅", uint64(test.Value), result, "==", test.Output)
			}
		})
	}

}

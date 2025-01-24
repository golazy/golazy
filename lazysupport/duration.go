package lazysupport

import (
	"fmt"
	"math"
	"time"
)

var day = time.Hour * 24
var Year = day * (365 + 365 + 365 + 366) / 4
var Month = Year / 12

// ShortDuration returns a short string representation of the duration regardless of the unit.
//
// For numbers below to one minute the rules are as follows:
//   - If the integer part is bigger than 100, it will hide the fractional part. For example: 120ms
//   - If the integer part is bigger than 10, it will show at max 1 decimal. For example: 12.3s
//   - If the integer part is bigger than 1, it will show at max 2 decimals. For example: 1.23s
//   - If the integer part is 0, it will use the next unit.
//   - If duration is 0 it is return as 0s.
//
// For numbers above or equal to one minute it will include at maximum two units. For example:
//   - 1y 2d
//   - 2m 3d
//   - 1h 30m
//   - 3m 2s
//
// If the second unit is 0 that part is omitted.
//
// The valid units are
//   - days (d)
//   - hours (h)
//   - minutes (m)
//   - seconds (s)
//   - milliseconds (ms)
//   - microseconds (µs)
//   - nanoseconds (ns)
//
// For months and years, it is used the average number of days in a month and year respectively: DaysInYear and DaysInMonth.
//
// It is advise to inform the user that the duration is approximate.
func ShortDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	formatValue := func(value float64, unit string) string {
		decimalPlaces := checkDecimalPlaces(value)
		switch {
		case value >= 100:
			return fmt.Sprintf("%.0f%s", value, unit)
		case value >= 10:
			if decimalPlaces == 0 {
				return fmt.Sprintf("%.0f%s", value, unit)
			}
			return fmt.Sprintf("%.1f%s", value, unit)
		default:
			if decimalPlaces == 0 {
				return fmt.Sprintf("%.0f%s", value, unit)
			} else if decimalPlaces == 1 {
				return fmt.Sprintf("%.1f%s", value, unit)
			}
			return fmt.Sprintf("%.2f%s", value, unit)
		}
	}

	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d)
	} else if d < time.Millisecond {
		return formatValue(float64(d)/float64(time.Microsecond), "µs")
	} else if d < time.Second {
		return formatValue(float64(d)/float64(time.Millisecond), "ms")
	} else if d < time.Minute {
		return formatValue(float64(d)/float64(time.Second), "s")
	} else if d < time.Hour {
		minutes := d / time.Minute
		seconds := (d % time.Minute) / time.Second
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else if d < day {
		hours := d / time.Hour
		minutes := (d % time.Hour) / time.Minute
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if d < Month {
		days := d / day
		hours := (d % day) / time.Hour
		if hours == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if d < Year {
		months := d / Month
		days := (d % Month) / day
		if days == 0 {
			return fmt.Sprintf("%dM", months)
		}
		return fmt.Sprintf("%dM %dd", months, days)
	} else {
		years := d / Year
		months := (d % Year) / Month
		if months == 0 {
			return fmt.Sprintf("%dy", years)
		}
		return fmt.Sprintf("%dy %dM", years, months)
	}
}

type Duration time.Duration

// String will call ShortDuration on the duration.
func (d Duration) String() string {
	return ShortDuration(time.Duration(d))
}

func checkDecimalPlaces(value float64) int {
	if value == float64(int(value)) {
		return 0
	}
	if math.Round(value*10)/10 == value {
		return 1
	}
	if math.Round(value*100)/100 == value {
		return 2
	}
	return -1
}

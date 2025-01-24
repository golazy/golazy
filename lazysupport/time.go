package lazysupport

import (
	"fmt"
	"time"
)

func TimeAgoInWords(t time.Time) string {
	return TimeDistanceInWords(time.Since(t)) + " ago"
}

func TimeDistanceInWords(d time.Duration) string {
	if d < 0 {
		d = d * -1
	}
	minutes := int(d.Minutes())
	hours := int(minutes / 60)
	days := int(d.Hours() / 24)
	months := int(d.Hours() / 24 / 30)
	years := int(months / 12)
	switch {
	case minutes < 1:
		return "less than a minute"

	case minutes == 1:
		return "a minute"

	case minutes < 45: // 45 minutes
		return fmt.Sprint(minutes, " minutes")

	case minutes < 90: // 1 hour and a half
		return "about an hour"

	case hours < 24:
		return fmt.Sprint(minutes/60, " hours")

	case hours < 48:
		return "a day"

	case days < 30: // 30 days
		return fmt.Sprint(days, " days")

	case months == 1 && days > 25:
		return fmt.Sprint("a month")

	case months < 12:
		return fmt.Sprint("about ", months, " months")

	case years < 2:
		return "more than a year"
	default:
		return fmt.Sprint("about ", months/12, " years")

	}

}

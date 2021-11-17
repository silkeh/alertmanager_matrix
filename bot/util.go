package bot

import (
	"regexp"
	"strconv"
	"time"
)

var durationRegex = regexp.MustCompile(`(\d+)(\w)`)

// Additional durations.
const (
	Day  = 24 * time.Hour
	Week = 7 * Day
	Year = 365 * Day
)

// parseDuration parses a duration in days weeks or years in addition to
// the times parsed by time.ParseDuration.
func parseDuration(s string) (time.Duration, error) {
	m := durationRegex.FindStringSubmatch(s)
	if m == nil {
		return time.ParseDuration(s)
	}

	i, err := strconv.Atoi(m[1])
	if err != nil {
		return time.ParseDuration(s)
	}

	switch m[2] {
	case `d`:
		return time.Duration(i) * Day, nil
	case `w`:
		return time.Duration(i) * Week, nil
	case `y`:
		return time.Duration(i) * Year, nil
	default:
		return time.ParseDuration(s)
	}
}

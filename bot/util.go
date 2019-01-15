package bot

import (
	"regexp"
	"strconv"
	"time"
)

var durationRegex = regexp.MustCompile(`(\d+)(\w)`)

// parseDuration parses a duration in days weeks or years in addition to
// the times parsed by time.ParseDuration
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
		return time.Duration(i*24) * time.Hour, nil
	case `w`:
		return time.Duration(i*24*7) * time.Hour, nil
	case `y`:
		return time.Duration(i*24*365) * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}

// shortCommand returns the n letter abbreviation of a command.
func shortCommand(cmd []string, n int) (str string) {
	for _, c := range cmd {
		if len(c) > 0 {
			str += string(c[0])
		}
		if len(str) == n {
			break
		}
	}
	return
}

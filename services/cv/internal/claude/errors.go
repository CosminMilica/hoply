package claude

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The CLI's diagnostics, as work/reviewer reads them (src/reviewer/llm/errors.py).
var (
	loginDead = regexp.MustCompile(`(?i)not logged in|failed to authenticate|oauth (access )?token (is invalid|has expired)|token[^\n]{0,20}revoked|invalid api key`)
	// The trailing \b keeps "limitation" out.
	usageLimit = regexp.MustCompile(`(?i)\b(session|usage|rate|weekly|daily|monthly)[ _-]?limits?\b|\b429 Too Many Requests\b|\bstatus:? 429\b`)
	resetTime  = regexp.MustCompile(`(?i)resets?\s+(\d{1,2})(?::(\d{2}))?\s*([ap]m)?\s*\(?\s*UTC\s*\)?`)
	resetDate  = regexp.MustCompile(`(?i)resets?\s+([A-Za-z]{3})[a-z]*\.?\s+(\d{1,2})(?:st|nd|rd|th)?,?\s+(\d{1,2})(?::(\d{2}))?\s*([ap]m)?\s*\(?\s*UTC\s*\)?`)
	months     = map[string]time.Month{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}
)

const (
	defaultPark = 15 * time.Minute
	maxPark     = 6 * time.Hour
)

func hour24(hour int, ampm string) int {
	switch strings.ToLower(ampm) {
	case "pm":
		if hour != 12 {
			return hour + 12
		}
	case "am":
		if hour == 12 {
			return 0
		}
	}
	return hour
}

// ParseReset is the time named by "resets 4:20pm (UTC)" or "resets Sep 20,
// 11pm (UTC)", the earlier of the two when both appear; zero when none does.
func ParseReset(text string, now time.Time) time.Time {
	now = now.UTC()
	var found []time.Time
	if m := resetDate.FindStringSubmatch(text); m != nil {
		if month, ok := months[strings.ToLower(m[1])]; ok {
			day, _ := strconv.Atoi(m[2])
			hour, _ := strconv.Atoi(m[3])
			minute, _ := strconv.Atoi(m[4])
			hour = hour24(hour, m[5])
			if hour <= 23 && minute <= 59 && day >= 1 && day <= 31 {
				at := time.Date(now.Year(), month, day, hour, minute, 0, 0, time.UTC)
				if at.Before(now.Add(-24 * time.Hour)) {
					at = at.AddDate(1, 0, 0)
				}
				found = append(found, at)
			}
		}
	}
	if m := resetTime.FindStringSubmatch(text); m != nil {
		hour, _ := strconv.Atoi(m[1])
		minute, _ := strconv.Atoi(m[2])
		hour = hour24(hour, m[3])
		if hour <= 23 && minute <= 59 {
			at := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
			if !at.After(now) {
				at = at.Add(24 * time.Hour)
			}
			found = append(found, at)
		}
	}
	var earliest time.Time
	for _, at := range found {
		if earliest.IsZero() || at.Before(earliest) {
			earliest = at
		}
	}
	return earliest
}

// ClampReset keeps a reset between a minute and six hours away; no reset is
// fifteen minutes away.
func ClampReset(until, now time.Time) time.Time {
	if until.IsZero() {
		until = now.Add(defaultPark)
	}
	if until.Before(now.Add(time.Minute)) {
		return now.Add(time.Minute)
	}
	if until.After(now.Add(maxPark)) {
		return now.Add(maxPark)
	}
	return until
}

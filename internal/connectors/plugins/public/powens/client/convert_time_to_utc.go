package client

import "time"

// ConvertTimeToUTC converts a date string to UTC, assuming it is in Europe/Paris
// Powens sends date fields in Europe/Paris timezone (without offset).
// Parse in that location and convert to UTC for a consistent internal representation.
func ConvertTimeToUTC(input string, format string) (time.Time, error) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.Time{}, err
	}

	tm, err := time.ParseInLocation(format, input, loc)
	if err != nil {
		return time.Time{}, err
	}
	return tm.UTC(), nil
}

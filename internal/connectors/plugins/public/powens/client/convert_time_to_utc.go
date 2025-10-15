package client

import "time"

// ConvertPowensTimeToUTC converts a date string to UTC, assuming it is in Europe/Paris
// Powens sends date fields in Europe/Paris timezone (without offset).
// Parse in that location and convert to UTC for a consistent internal representation.
func ConvertPowensTimeToUTC(input string, format string) (time.Time, error) {
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

// ConvertUTCToPowensTime converts a date in UTC to Powens string format, while making sure
// that format is in Europe/Paris timezone (without offset).
// Powens expects naive local time strings (Europe/Paris) like "2006-01-02 15:04:05".
// We therefore convert the UTC time to Europe/Paris and format it without any offset.
func ConvertUTCToPowensTime(input time.Time, format string) (string, error) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return "", err
	}
	paris := input.In(loc)
	return paris.Format(format), nil
}

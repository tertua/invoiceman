package utils

import (
	"time"
)

// DateLayout is the single date format exchanged with the frontend (YYYY-MM-DD).
// Keep all parsing/formatting here so app/controllers stays consistent and
// frontend parsing in web/src stays aligned.
const DateLayout = "2006-01-02"

// ParseDate parses an optional YYYY-MM-DD date string.
// Empty string returns (nil, nil) for optional fields.
func ParseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(DateLayout, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseRequiredDate parses a mandatory YYYY-MM-DD date string.
func ParseRequiredDate(value string) (time.Time, error) {
	parsed, err := time.Parse(DateLayout, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

// FormatDate formats an optional date for the frontend ("" when nil).
func FormatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(DateLayout)
}

// FormatTime formats a concrete date value for the frontend.
func FormatTime(value time.Time) string {
	return value.Format(DateLayout)
}

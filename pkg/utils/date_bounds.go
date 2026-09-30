package utils

import "time"

// DateOnly returns the calendar date of t, read in t's own location, as UTC
// midnight.
//
// Date-only columns (issue_date, due_date, paid_on, expense_date) hold the
// calendar date the API handed in (ParseDate/ParseRequiredDate), so they always
// sit at UTC midnight. SQL bounds over them must be built the same way, because
// the two dialects disagree otherwise: SQLite stores the value as text and
// compares lexicographically, so the UTC offset decides any tie, while
// PostgreSQL compares instants, so a local-zone bound shifts the whole window
// by the offset. UTC midnight on both sides keeps the dialects aligned whatever
// the server timezone is.
func DateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// MonthBounds returns [start, next) for the calendar month containing t, as
// UTC midnights: start is inclusive, next exclusive. The month comes from t's
// own calendar (its location), so "this month" follows the server clock the
// same way callers and tests spell out a date.
func MonthBounds(t time.Time) (start, next time.Time) {
	start = DateOnly(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()))
	return start, start.AddDate(0, 1, 0)
}

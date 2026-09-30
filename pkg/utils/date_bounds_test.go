package utils

import (
	"testing"
	"time"
)

// utc7 is a zone east of UTC: its local date runs ahead of UTC for the first
// hours of every day, which is where the month-bound bug showed up.
var utc7 = time.FixedZone("UTC+07", 7*60*60)

func TestDateOnlyKeepsTheCalendarDateInUTC(t *testing.T) {
	// Local 01:00 on the 1st is still the previous day in UTC; the calendar
	// date must win, not the instant.
	got := DateOnly(time.Date(2026, 10, 1, 1, 0, 0, 0, utc7))
	want := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("DateOnly = %v, want %v", got, want)
	}
	if got.Location() != time.UTC {
		t.Errorf("DateOnly location = %v, want UTC", got.Location())
	}
	if again := DateOnly(got); !again.Equal(got) {
		t.Errorf("DateOnly is not idempotent: %v then %v", got, again)
	}
}

func TestMonthBoundsCoverTheLocalCalendarMonth(t *testing.T) {
	start, next := MonthBounds(time.Date(2026, 10, 1, 1, 0, 0, 0, utc7))
	wantStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	wantNext := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", start, wantStart)
	}
	if !next.Equal(wantNext) {
		t.Errorf("next = %v, want %v", next, wantNext)
	}
	if start.Location() != time.UTC || next.Location() != time.UTC {
		t.Errorf("bounds must be UTC, got %v and %v", start.Location(), next.Location())
	}
}

func TestMonthBoundsAcceptAnyZone(t *testing.T) {
	// A moment in the last hours of a month east of UTC, and one west of it,
	// must both land in their local calendar month.
	utcMinus7 := time.FixedZone("UTC-07", -7*60*60)
	cases := []struct {
		name  string
		at    time.Time
		month time.Month
	}{
		// Local 1 Oct 00:30 is still 30 Sep in UTC.
		{"east just after midnight", time.Date(2026, 10, 1, 0, 30, 0, 0, utc7), time.October},
		// Local 30 Sep 23:30 is already 1 Oct in UTC.
		{"west just before midnight", time.Date(2026, 9, 30, 23, 30, 0, 0, utcMinus7), time.September},
	}
	for _, tc := range cases {
		start, next := MonthBounds(tc.at)
		if start.Month() != tc.month || next.Month() != tc.month+1 {
			t.Errorf("%s: bounds %v..%v, want month %v", tc.name, start, next, tc.month)
		}
		if start.Location() != time.UTC || next.Location() != time.UTC {
			t.Errorf("%s: bounds must be UTC, got %v and %v", tc.name, start.Location(), next.Location())
		}
	}
}

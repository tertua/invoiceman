package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tertua/tupay/app/queries"
)

// inviteTTL: zero or negative hours hand the default (7 days) back to the query, positive hours become a duration.
func TestInviteTTL(t *testing.T) {
	cases := []struct {
		hours int
		want  time.Duration
	}{
		{0, 0},
		{-2, 0},
		{1, time.Hour},
		{24 * 7, 168 * time.Hour},
	}
	for _, tc := range cases {
		if got := inviteTTL(tc.hours); got != tc.want {
			t.Errorf("inviteTTL(%d) = %v, want %v", tc.hours, got, tc.want)
		}
	}
}

// inviteTokenStatus: unknown, expired, revoked and already-used tokens answer 400 with the stable message; anything else is a 500.
func TestInviteTokenStatus(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		message string
		ok      bool
	}{
		{"unknown token is reported as expired", sql.ErrNoRows, 400, "invite expired", true},
		{"wrapped not-found is reported as expired", fmt.Errorf("load: %w", sql.ErrNoRows), 400, "invite expired", true},
		{"expired invite keeps the query message", queries.ErrInviteExpired, 400, "invite expired", true},
		{"revoked invite keeps the query message", queries.ErrInviteRevoked, 400, "invite revoked", true},
		{"accepted invite keeps the query message", queries.ErrInviteAccepted, 400, "invite already accepted", true},
		{"unexpected failure is not a client error", errors.New("driver exploded"), 0, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, message, ok := inviteTokenStatus(tc.err)
			if status != tc.status || message != tc.message || ok != tc.ok {
				t.Fatalf("inviteTokenStatus(%v) = (%d, %q, %v), want (%d, %q, %v)", tc.err, status, message, ok, tc.status, tc.message, tc.ok)
			}
		})
	}
}

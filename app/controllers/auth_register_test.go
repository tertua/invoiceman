package controllers

import "testing"

// TestRegistrationOpen pins the invite-only gate condition: open installs and the first account always pass, and a closed install with users needs a token to pass the coarse check (validity is checked later).
func TestRegistrationOpen(t *testing.T) {
	cases := []struct {
		name        string
		allow       bool
		userCount   int64
		inviteToken string
		want        bool
	}{
		{"allowed with users", true, 5, "", true},
		{"allowed without users", true, 0, "", true},
		{"first account bootstraps while disabled", false, 0, "", true},
		{"disabled with users needs a token", false, 5, "", false},
		{"disabled with users and a token passes the coarse gate", false, 5, "some-token", true},
		{"disabled with a token but no users still passes", false, 0, "some-token", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := registrationOpen(tt.allow, tt.userCount, tt.inviteToken); got != tt.want {
				t.Errorf("registrationOpen(%v, %d, %q) = %v, want %v", tt.allow, tt.userCount, tt.inviteToken, got, tt.want)
			}
		})
	}
}

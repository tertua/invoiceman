package controllers

import (
	"testing"

	"github.com/tertua/tupay/app/models"
)

// TestVerifyEmailGate pins the pending-status rule: verification on with an
// existing install makes new accounts pending, the first-install bootstrap and
// the flag-off path stay active.
func TestVerifyEmailGate(t *testing.T) {
	cases := []struct {
		name    string
		require string
		count   int64
		want    int
	}{
		{"required with users is pending", "true", 5, models.UserStatusPending},
		{"required bootstrap stays active", "true", 0, models.UserStatusActive},
		{"flag off is active", "false", 5, models.UserStatusActive},
		{"flag off bootstrap is active", "false", 0, models.UserStatusActive},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REQUIRE_EMAIL_VERIFICATION", tt.require)
			if got := verifyEmailGate(tt.count); got != tt.want {
				t.Errorf("verifyEmailGate(%d) with flag %q = %d, want %d", tt.count, tt.require, got, tt.want)
			}
		})
	}
}

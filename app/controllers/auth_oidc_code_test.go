package controllers

import (
	"errors"
	"testing"
)

// TestOIDCResolveCode pins the sentinel -> redirect-code mapping so a busy or
// unknown failure never leaks a provider detail and always reads "busy".
func TestOIDCResolveCode(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errOIDCEmailUnverified, "email"},
		{errOIDCDenied, "denied"},
		{errOIDCBusy, "busy"},
		{errors.New("some other failure"), "busy"},
	}
	for _, tc := range cases {
		if got := oidcResolveCode(tc.err); got != tc.want {
			t.Errorf("oidcResolveCode(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

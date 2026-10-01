package middleware

import "testing"

// TestSecureEqual pins the constant-time comparison helper's truth table.
func TestSecureEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"a", "a", true},
		{"a", "b", false},
		{"", "", true},
		{"", "a", false},
		{"abc", "abcd", false},
		{"token-xyz", "token-xyz", true},
	}
	for _, tc := range cases {
		if got := secureEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("secureEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

package middleware

import "crypto/subtle"

// secureEqual compares two strings in constant time. Inputs are compared as raw
// bytes; ConstantTimeCompare returns 0 immediately for differing lengths, which
// is still safe (length is not a secret here) and avoids leaking how far a
// guess matched.
func secureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

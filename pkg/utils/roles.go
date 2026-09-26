package utils

// HasRole reports whether the role is one of the allowed roles.
func HasRole(role string, allowed ...string) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

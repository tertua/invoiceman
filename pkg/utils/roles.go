package utils

import (
	"fmt"

	"github.com/tertua/invoiceman/pkg/repository"
)

// VerifyRole func for verifying a given role.
func VerifyRole(role string) (string, error) {
	// Switch given role.
	switch role {
	case repository.AdminRoleName:
		// Nothing to do, verified successfully.
	case repository.ModeratorRoleName:
		// Nothing to do, verified successfully.
	case repository.UserRoleName:
		// Nothing to do, verified successfully.
	default:
		// Return error message.
		return "", fmt.Errorf("role '%v' does not exist", role)
	}

	return role, nil
}

// HasRole reports whether the role is one of the allowed roles.
func HasRole(role string, allowed ...string) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

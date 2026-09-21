package models

// RoleInput describes an admin role assignment request.
type RoleInput struct {
	Role string `json:"role" validate:"required,oneof=user moderator"`
}

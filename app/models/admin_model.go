package models

// MigrateDownInput describes an admin schema-rollback request. Rollbacks
// destroy schema (never data rows, but dropped columns lose their values),
// so the caller must opt in twice: an in-range target plus confirm=true.
type MigrateDownInput struct {
	TargetVersion int  `json:"target_version" validate:"required,min=1"`
	Confirm       bool `json:"confirm" validate:"eq=true"`
}

// RoleInput is the admin role-change payload; only the two platform roles are accepted.
type RoleInput struct {
	Role string `json:"role" validate:"required,oneof=user admin"`
}

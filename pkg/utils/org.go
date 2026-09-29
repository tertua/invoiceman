package utils

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Context locals keys set by the OrgContext middleware.
const (
	SessionOrgIDKey   = "orgID"
	SessionOrgRoleKey = "orgRole"
)

// ErrOrgNotMember reports a request context carrying no organization membership.
var ErrOrgNotMember = errors.New("org membership required")

// ErrOrgOwnerRequired reports a non-owner role on an owner-only action.
var ErrOrgOwnerRequired = errors.New("org owner role required")

// CurrentOrgID returns the organization ID stored by the OrgContext middleware.
func CurrentOrgID(c fiber.Ctx) (uuid.UUID, error) {
	orgID, ok := c.Locals(SessionOrgIDKey).(uuid.UUID)
	if !ok || orgID == uuid.Nil {
		return uuid.Nil, ErrOrgNotMember
	}
	return orgID, nil
}

// CurrentOrgRole returns the organization role ("owner" or "staff") stored by the OrgContext middleware.
func CurrentOrgRole(c fiber.Ctx) (string, error) {
	role, ok := c.Locals(SessionOrgRoleKey).(string)
	if !ok || role == "" {
		return "", ErrOrgNotMember
	}
	return role, nil
}

// OrgIDFromLocals returns the organization ID, or uuid.Nil when absent, for best-effort callers like recordAudit.
func OrgIDFromLocals(c fiber.Ctx) uuid.UUID {
	orgID, _ := c.Locals(SessionOrgIDKey).(uuid.UUID)
	return orgID
}

// CurrentActorID returns the authenticated user identity recorded by AuthRequired for audit trails.
func CurrentActorID(c fiber.Ctx) uuid.UUID {
	userID, _ := c.Locals(SessionUserIDKey).(uuid.UUID)
	return userID
}

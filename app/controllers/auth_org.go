package controllers

import (
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// loginOrgHint resolves the active org for a fresh session; best-effort, so any failure yields "" and never blocks the login.
func loginOrgHint(db *database.Queries, userID uuid.UUID) string {
	orgID, err := db.ResolveActiveOrgID(userID, uuid.Nil)
	if err != nil {
		return ""
	}
	return orgID.String()
}

// orgPayload builds the org block of /auth/me (active org + every membership with its role); best-effort, any failure yields an empty map.
func orgPayload(db *database.Queries, userID uuid.UUID) fiber.Map {
	orgID, err := db.ResolveActiveOrgID(userID, uuid.Nil)
	if err != nil {
		return fiber.Map{}
	}
	org, err := db.GetOrg(orgID)
	if err != nil {
		return fiber.Map{}
	}
	role, err := db.GetRole(orgID, userID)
	if err != nil {
		return fiber.Map{}
	}
	rows, err := db.ListByUser(userID)
	if err != nil {
		return fiber.Map{}
	}
	memberships := make([]fiber.Map, 0, len(rows))
	for _, m := range rows {
		memberOrg, err := db.GetOrg(m.OrgID)
		if err != nil {
			return fiber.Map{}
		}
		memberships = append(memberships, fiber.Map{"org_id": m.OrgID, "name": memberOrg.Name, "role": m.Role})
	}
	return fiber.Map{"id": org.ID, "name": org.Name, "role": role, "memberships": memberships}
}

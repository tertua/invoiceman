package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// loginOrgHint resolves the active org for a fresh session; best-effort, so any failure yields "" and never blocks the login.
func loginOrgHint(db *database.Queries, userID uuid.UUID) string {
	orgID, err := db.ResolveActiveOrgID(userID, uuid.Nil)
	if err != nil {
		return ""
	}
	return orgID.String()
}

// activeOrgHint reads the org OrgContext resolved for this request; a miss hands back uuid.Nil so resolution falls back to the session/default path.
func activeOrgHint(c fiber.Ctx) uuid.UUID {
	if orgID, err := utils.CurrentOrgID(c); err == nil {
		return orgID
	}
	return uuid.Nil
}

// orgPayload builds the org block of /auth/me (active org + every membership with its role); best-effort, any failure yields an empty map. hint is the middleware-resolved active org so the reported role matches the org actually in use, not the oldest membership.
func orgPayload(db *database.Queries, userID, hint uuid.UUID) any {
	orgID, err := db.ResolveActiveOrgID(userID, hint)
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
	memberships := make([]orgMembershipRow, 0, len(rows))
	for _, m := range rows {
		memberOrg, err := db.GetOrg(m.OrgID)
		if err != nil {
			return fiber.Map{}
		}
		memberships = append(memberships, orgMembershipRow{OrgID: m.OrgID, Name: memberOrg.Name, Role: m.Role})
	}
	return orgPayloadResponse{ID: org.ID, Name: org.Name, Role: role, Memberships: memberships}
}

type orgMembershipRow struct {
	OrgID uuid.UUID `json:"org_id"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

type orgPayloadResponse struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	Role        string             `json:"role"`
	Memberships []orgMembershipRow `json:"memberships"`
}

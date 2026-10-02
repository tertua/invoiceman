package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/app/queries"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// orgInviteBody is the optional POST /orgs/invites payload; absent fields fall back to defaults (role staff, TTL 7 days).
type orgInviteBody struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	TTLHours int    `json:"ttl_hours"`
}

// orgAcceptBody is the POST /orgs/invites/accept payload.
type orgAcceptBody struct {
	Token string `json:"token"`
}

// normalizeInviteRole resolves the requested invite role: empty means staff; owner/staff pass through; anything else is refused with ok=false.
func normalizeInviteRole(raw string) (string, bool) {
	switch role := strings.TrimSpace(raw); role {
	case "":
		return models.RoleStaff, true
	case models.RoleOwner, models.RoleStaff:
		return role, true
	default:
		return "", false
	}
}

// inviteRole resolves the role an invite grants, falling back to staff for rows minted before roles were stored.
func inviteRole(invite models.OrgInvite) string {
	if invite.Role == models.RoleOwner || invite.Role == models.RoleStaff {
		return invite.Role
	}
	return models.RoleStaff
}

// inviteTTL converts the requested hours to a duration; 0 or less hands the default (7 days) back to the query.
func inviteTTL(hours int) time.Duration {
	if hours <= 0 {
		return 0
	}
	return time.Duration(hours) * time.Hour
}

// inviteTokenStatus maps a GetValidByToken failure onto the stable client message; ok is false for unexpected errors the caller must answer with a 500.
func inviteTokenStatus(err error) (status int, message string, ok bool) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fiber.StatusBadRequest, "invite expired", true
	case errors.Is(err, queries.ErrInviteExpired), errors.Is(err, queries.ErrInviteRevoked), errors.Is(err, queries.ErrInviteAccepted):
		return fiber.StatusBadRequest, err.Error(), true
	default:
		return 0, "", false
	}
}

// queueInviteMail enqueues the invite notice best-effort: a failure only logs, because the link the owner shares already works (D9).
func queueInviteMail(c fiber.Ctx, db *database.Queries, orgID uuid.UUID, to, token string) {
	org, err := db.GetOrg(orgID)
	if err != nil {
		utils.RequestLogger(c).Warn("invite mail skipped", "err", err)
		return
	}
	cfg := configs.Get()
	link := strings.TrimRight(cfg.Mail.AppPublicURL, "/") + "/invite/" + token
	body := fmt.Sprintf("You have been invited to join %s on %s.\n\nOpen this link to join the team:\n%s\n\nIt stops working once it expires, is revoked or someone else uses it.", org.Name, cfg.AppName, link)
	if err := db.EnqueueMail(&models.MailOutbox{To: to, Subject: "Join " + org.Name + " on " + cfg.AppName, Body: body}); err != nil {
		utils.RequestLogger(c).Warn("invite mail queue failed", "email", to, "err", err)
	}
}

// ListMembers returns the active org's memberships as user_id+role pairs (no batch user query exists, so display names stay off the payload).
// @Description List the members of the active organization.
// @Summary list organization members
// @Tags Org
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/members [get]
func ListMembers(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	rows, err := db.ListByOrg(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load members", nil)
	}
	members := make([]fiber.Map, 0, len(rows))
	for _, m := range rows {
		members = append(members, fiber.Map{"user_id": m.UserID, "role": m.Role})
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"members": members})
}

type orgInviteRow struct {
	ID        uuid.UUID `json:"id"`
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	URL       string    `json:"url"`
}

// CreateOrgInvite mints a single-use join link; the route already guards the owner role and the same check repeats here (defence in depth).
// @Description Create an invite link for the active organization (owner only).
// @Summary create organization invite
// @Tags Org
// @Accept json
// @Produce json
// @Param request body object false "Optional invite payload: email (string) and ttl_hours (int, 0 means 7 days)"
// @Success 201 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/invites [post]
func CreateOrgInvite(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	role, err := utils.CurrentOrgRole(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	if role != models.RoleOwner {
		return utils.Fail(c, fiber.StatusForbidden, "org.ownerRequired", nil)
	}
	input := &orgInviteBody{}
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(input); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
		}
	}
	var email *string
	if trimmed := strings.TrimSpace(input.Email); trimmed != "" {
		email = &trimmed
	}
	joinRole, ok := normalizeInviteRole(input.Role)
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invite role", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	invite, err := db.CreateInvite(orgID, email, joinRole, inviteTTL(input.TTLHours))
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create invite", nil)
	}
	recordAudit(c, db, utils.CurrentActorID(c), "org.invite.create", "org_invite", invite.ID.String(), "")
	if email != nil && configs.Get().Mail.SMTPHost != "" {
		queueInviteMail(c, db, orgID, *email, invite.Token)
	}
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"invite": orgInviteRow{
		ID:        invite.ID,
		Token:     invite.Token,
		Role:      invite.Role,
		ExpiresAt: invite.ExpiresAt,
		URL:       "/invite/" + invite.Token,
	}})
}

// ListOrgInvites returns the org's invites newest first; the route keeps it owner-only.
// @Description List invite links of the active organization (owner only).
// @Summary list organization invites
// @Tags Org
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/invites [get]
func ListOrgInvites(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	invites, err := db.ListInvitesByOrg(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invites", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"invites": invites})
}

// RevokeOrgInvite stamps one invite of the org revoked; the route keeps it owner-only.
// @Description Revoke an invite link of the active organization (owner only).
// @Summary revoke organization invite
// @Tags Org
// @Produce json
// @Param id path string true "Invite ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/invites/{id} [delete]
func RevokeOrgInvite(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invite id", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	if err := db.Revoke(orgID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invite not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to revoke invite", nil)
	}
	recordAudit(c, db, utils.CurrentActorID(c), "org.invite.revoke", "org_invite", id.String(), "")
	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "invite revoked"})
}

// AcceptOrgInvite redeems a link token: it joins the caller to the invite's org as staff, marks the invite used and makes that org the session's active org; an existing membership keeps its role (idempotent).
// @Description Accept an organization invite link; any signed-in user may redeem it.
// @Summary accept organization invite
// @Tags Org
// @Accept json
// @Produce json
// @Param request body object true "Invite payload with token"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/invites/accept [post]
func AcceptOrgInvite(c fiber.Ctx) error {
	input := &orgAcceptBody{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	token := strings.TrimSpace(input.Token)
	if token == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "token is required", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	invite, err := db.GetValidByToken(token)
	if err != nil {
		if status, message, ok := inviteTokenStatus(err); ok {
			return utils.Fail(c, status, message, nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invite", nil)
	}
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	// Claim before joining: single-use must hold under concurrent redemptions (the loser gets 400 without a membership).
	if err := db.MarkAccepted(invite.ID, userID); err != nil {
		if status, message, ok := inviteTokenStatus(err); ok {
			return utils.Fail(c, status, message, nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invite", nil)
	}
	orgID := invite.OrgID
	role, err := db.GetRole(orgID, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load membership", nil)
		}
		joinRole := inviteRole(invite)
		if cerr := db.CreateIfAbsent(orgID, userID, joinRole); cerr != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to join organization", nil)
		}
		role = joinRole
	}
	if err := persistActiveOrg(c, userID, orgID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to persist session", nil)
	}
	org, err := db.GetOrg(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load organization", nil)
	}
	recordAudit(c, db, userID, "org.invite.accept", "org_invite", invite.ID.String(), "")
	return utils.OK(c, fiber.StatusOK, fiber.Map{"org": fiber.Map{"id": org.ID.String(), "name": org.Name}, "role": role})
}

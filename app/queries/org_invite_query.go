package queries

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// Stable invite state errors so controllers can map 400/404 without re-reading the row (design: expired 400, revoked 404).
var (
	ErrInviteExpired  = errors.New("invite expired")
	ErrInviteRevoked  = errors.New("invite revoked")
	ErrInviteAccepted = errors.New("invite already accepted")
)

// OrgInviteQueries mints and validates single-use org invite tokens (link-only channel, D9).
type OrgInviteQueries struct {
	*gorm.DB
}

// CreateInvite mints a 64-hex token valid for ttl (zero falls back to models.OrgInviteDefaultTTL); email is optional.
func (q *OrgInviteQueries) CreateInvite(orgID uuid.UUID, email *string, ttl time.Duration) (models.OrgInvite, error) {
	if ttl <= 0 {
		ttl = models.OrgInviteDefaultTTL
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return models.OrgInvite{}, err
	}
	invite := models.OrgInvite{
		ID: uuid.New(), OrgID: orgID, Token: hex.EncodeToString(raw), Role: models.RoleStaff,
		ExpiresAt: time.Now().Add(ttl), CreatedAt: time.Now(),
	}
	if email != nil {
		invite.Email = *email
	}
	if err := q.Create(&invite).Error; err != nil {
		return models.OrgInvite{}, err
	}
	return invite, nil
}

// GetValidByToken loads a token and rejects revoked, expired, or accepted invites with a stable sentinel; the row rides along so a repeat accept can stay idempotent.
func (q *OrgInviteQueries) GetValidByToken(token string) (models.OrgInvite, error) {
	invite := models.OrgInvite{}
	if err := q.Where("token = ?", token).First(&invite).Error; err != nil {
		return invite, notFound(err)
	}
	switch {
	case invite.RevokedAt != nil:
		return invite, ErrInviteRevoked
	case !invite.ExpiresAt.After(time.Now()):
		return invite, ErrInviteExpired
	case invite.AcceptedBy != nil:
		return invite, ErrInviteAccepted
	}
	return invite, nil
}

// MarkAccepted claims the invite for userID; a repeat or concurrent claim answers ErrInviteAccepted so single-use holds even when two redemptions race.
func (q *OrgInviteQueries) MarkAccepted(id, userID uuid.UUID) error {
	invite := models.OrgInvite{}
	if err := q.Where("id = ?", id).First(&invite).Error; err != nil {
		return notFound(err)
	}
	if invite.AcceptedBy != nil {
		return ErrInviteAccepted
	}
	res := q.Model(&models.OrgInvite{}).Where("id = ? AND accepted_by IS NULL", id).Update("accepted_by", userID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInviteAccepted
	}
	return nil
}

// Revoke stamps revoked_at on one invite of the org; a miss returns sql.ErrNoRows so callers can 404.
func (q *OrgInviteQueries) Revoke(orgID, id uuid.UUID) error {
	now := time.Now()
	res := q.Model(&models.OrgInvite{}).Where("id = ? AND org_id = ?", id, orgID).Update("revoked_at", &now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListInvitesByOrg returns the org's invites newest first; named apart from MembershipQueries.ListByOrg because both embed in database.Queries.
func (q *OrgInviteQueries) ListInvitesByOrg(orgID uuid.UUID) ([]models.OrgInvite, error) {
	out := []models.OrgInvite{}
	if err := q.Where("org_id = ?", orgID).Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

package queries

import (
	"errors"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// MembershipQueries reads and writes org memberships (owner/staff roles).
type MembershipQueries struct {
	*gorm.DB
}

// ListByUser returns every org the user belongs to, oldest membership first (D3 resolution input).
func (q *MembershipQueries) ListByUser(userID uuid.UUID) ([]models.Membership, error) {
	out := []models.Membership{}
	if err := q.Where("user_id = ?", userID).Order("created_at ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// GetRole returns the user's role in an org, or sql.ErrNoRows when the user is not a member.
func (q *MembershipQueries) GetRole(orgID, userID uuid.UUID) (string, error) {
	m := models.Membership{}
	if err := q.Where("org_id = ? AND user_id = ?", orgID, userID).First(&m).Error; err != nil {
		return "", notFound(err)
	}
	return m.Role, nil
}

// CountByUser returns how many organizations the user belongs to.
func (q *MembershipQueries) CountByUser(userID uuid.UUID) (int64, error) {
	var count int64
	if err := q.Model(&models.Membership{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CreateOwner inserts the founding owner membership.
func (q *MembershipQueries) CreateOwner(orgID, userID uuid.UUID) error {
	return q.create(orgID, userID, models.RoleOwner)
}

// AddStaff inserts a staff membership, used when an invite is accepted.
func (q *MembershipQueries) AddStaff(orgID, userID uuid.UUID) error {
	return q.create(orgID, userID, models.RoleStaff)
}

// CreateIfAbsent joins a user to an org only when no membership exists yet; a lost race still reports success (idempotent).
func (q *MembershipQueries) CreateIfAbsent(orgID, userID uuid.UUID, role string) error {
	existing := models.Membership{}
	err := q.Where("org_id = ? AND user_id = ?", orgID, userID).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := q.create(orgID, userID, role); err != nil {
		var winner models.Membership
		if rerr := q.Where("org_id = ? AND user_id = ?", orgID, userID).First(&winner).Error; rerr == nil {
			return nil
		}
		return err
	}
	return nil
}

// ListByOrg returns every member of an org, oldest first; org notification fan-out iterates this list (D11).
func (q *MembershipQueries) ListByOrg(orgID uuid.UUID) ([]models.Membership, error) {
	out := []models.Membership{}
	if err := q.Where("org_id = ?", orgID).Order("created_at ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Remove deletes one membership (member removal or org exit).
func (q *MembershipQueries) Remove(orgID, userID uuid.UUID) error {
	return q.Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&models.Membership{}).Error
}

// create inserts one membership with a fresh primary key.
func (q *MembershipQueries) create(orgID, userID uuid.UUID, role string) error {
	return q.Create(&models.Membership{ID: uuid.New(), OrgID: orgID, UserID: userID, Role: role}).Error
}

package queries

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// OrgQueries creates and resolves organizations, the tenant of the multi-user model.
type OrgQueries struct {
	*gorm.DB
}

// CreateOrg inserts a new organization and returns the stored row.
func (q *OrgQueries) CreateOrg(name string) (models.Organization, error) {
	org := models.Organization{ID: uuid.New(), Name: strings.TrimSpace(name)}
	if err := q.Create(&org).Error; err != nil {
		return models.Organization{}, err
	}
	return org, nil
}

// CreateOrgWithOwner inserts a new organization and its founding owner membership in one transaction (admin cross-tenant bootstrap); either both rows land or neither does.
func (q *OrgQueries) CreateOrgWithOwner(name string, ownerID uuid.UUID) (models.Organization, error) {
	var org models.Organization
	err := q.Transaction(func(tx *gorm.DB) error {
		created, err := (&OrgQueries{DB: tx}).CreateOrg(name)
		if err != nil {
			return err
		}
		if err := (&MembershipQueries{DB: tx}).CreateOwner(created.ID, ownerID); err != nil {
			return err
		}
		org = created
		return nil
	})
	if err != nil {
		return models.Organization{}, err
	}
	return org, nil
}

// GetOrg loads one organization by id, translating a miss to sql.ErrNoRows.
func (q *OrgQueries) GetOrg(id uuid.UUID) (models.Organization, error) {
	org := models.Organization{}
	if err := q.Where("id = ?", id).First(&org).Error; err != nil {
		return org, notFound(err)
	}
	return org, nil
}

// RenameOrg renames an organization; the owner-only check lives at the route layer.
func (q *OrgQueries) RenameOrg(orgID uuid.UUID, name string) error {
	now := time.Now()
	return q.Model(&models.Organization{}).Where("id = ?", orgID).Updates(map[string]any{
		"name": strings.TrimSpace(name), "updated_at": &now,
	}).Error
}

// personalOrgName derives the default tenant name from the owner's display name, then the email local part.
func personalOrgName(user models.User) string {
	if name := strings.TrimSpace(user.Name); name != "" {
		return name
	}
	if at := strings.Index(user.Email, "@"); at > 0 {
		return user.Email[:at]
	}
	return "Personal"
}

// EnsurePersonalOrg returns the org the user already belongs to, otherwise provisions an org plus owner membership in one transaction (idempotent).
func (q *OrgQueries) EnsurePersonalOrg(userID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := q.Transaction(func(tx *gorm.DB) error {
		memberships, err := (&MembershipQueries{DB: tx}).ListByUser(userID)
		if err != nil {
			return err
		}
		if len(memberships) > 0 {
			orgID = memberships[0].OrgID
			return nil
		}
		user, err := (&UserQueries{DB: tx}).GetUserByID(userID)
		if err != nil {
			return err
		}
		org, err := (&OrgQueries{DB: tx}).CreateOrg(personalOrgName(user))
		if err != nil {
			return err
		}
		if err := (&MembershipQueries{DB: tx}).CreateOwner(org.ID, userID); err != nil {
			return err
		}
		orgID = org.ID
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return orgID, nil
}

// ResolveActiveOrgID applies D3: a hinted org wins when the membership exists, otherwise auto-provision or take the oldest membership.
func (q *OrgQueries) ResolveActiveOrgID(userID, hint uuid.UUID) (uuid.UUID, error) {
	if hint != uuid.Nil {
		var membership models.Membership
		err := q.Where("org_id = ? AND user_id = ?", hint, userID).First(&membership).Error
		switch {
		case err == nil:
			return membership.OrgID, nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return uuid.Nil, err
		}
	}
	memberships, err := (&MembershipQueries{DB: q.DB}).ListByUser(userID)
	if err != nil {
		return uuid.Nil, err
	}
	if len(memberships) == 0 {
		return q.EnsurePersonalOrg(userID)
	}
	// Oldest first, so index 0 covers both the single and multi membership branches.
	return memberships[0].OrgID, nil
}

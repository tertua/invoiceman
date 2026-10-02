package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// UserIdentityQueries struct for queries from UserIdentity model.
type UserIdentityQueries struct {
	*gorm.DB
}

// GetByIdentity returns the identity row for an external (provider, sub) pair.
func (q *UserIdentityQueries) GetByIdentity(provider, sub string) (models.UserIdentity, error) {
	identity := models.UserIdentity{}
	if err := q.Where("provider = ? AND sub = ?", provider, sub).First(&identity).Error; err != nil {
		return identity, notFound(err)
	}
	return identity, nil
}

// CreateIdentity stores a new identity link.
func (q *UserIdentityQueries) CreateIdentity(identity *models.UserIdentity) error {
	return DoRetry(func() error {
		return q.Transaction(func(tx *gorm.DB) error {
			return tx.Create(identity).Error
		})
	})
}

// CreateUserTx inserts a user inside a caller-owned transaction.
func CreateUserTx(tx *gorm.DB, u *models.User) error {
	return tx.Create(u).Error
}

// ProvisionOIDCUser atomically creates a password-less user, its identity link, personal org and settings.
func (q *UserIdentityQueries) ProvisionOIDCUser(user *models.User, provider, sub, email string) error {
	return DoRetry(func() error {
		return q.Transaction(func(tx *gorm.DB) error {
			if err := CreateUserTx(tx, user); err != nil {
				return err
			}
			identity := &models.UserIdentity{
				ID: uuid.New(), UserID: user.ID, Provider: provider, Sub: sub, Email: email, CreatedAt: time.Now(),
			}
			if err := tx.Create(identity).Error; err != nil {
				return err
			}
			orgID, err := EnsurePersonalOrgTx(tx, user.ID)
			if err != nil {
				return err
			}
			return CreateSettingsTx(tx, models.DefaultSettings(orgID))
		})
	})
}

// LinkIdentity links an external identity to an existing user (auto-link by
// verified email). The caller has already checked that no row exists.
func (q *UserIdentityQueries) LinkIdentity(userID uuid.UUID, provider, sub, email string) error {
	identity := &models.UserIdentity{
		ID:        uuid.New(),
		UserID:    userID,
		Provider:  provider,
		Sub:       sub,
		Email:     email,
		CreatedAt: time.Now(),
	}
	return q.CreateIdentity(identity)
}

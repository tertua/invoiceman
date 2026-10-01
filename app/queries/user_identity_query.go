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

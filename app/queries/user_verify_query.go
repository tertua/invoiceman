package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// UpdateUserStatus query for changing a user's account status.
func (q *UserQueries) UpdateUserStatus(id uuid.UUID, status int) error {
	return q.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"updated_at":  time.Now(),
		"user_status": status,
	}).Error
}

// CreateEmailVerification query for storing an email verification token.
func (q *UserQueries) CreateEmailVerification(userID uuid.UUID, token string, expiresAt time.Time) error {
	// Send query to database.
	ev := &models.EmailVerification{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	if err := q.Create(ev).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

// GetEmailVerification query for getting an email verification token.
func (q *UserQueries) GetEmailVerification(token string) (models.EmailVerification, error) {
	// Define email verification variable.
	ev := models.EmailVerification{}

	// Send query to database.
	if err := q.Where("token = ?", token).First(&ev).Error; err != nil {
		// Return empty object and error.
		return ev, notFound(err)
	}

	// Return query result.
	return ev, nil
}

// DeleteEmailVerificationsByUser query for deleting all verification tokens of a user.
func (q *UserQueries) DeleteEmailVerificationsByUser(userID uuid.UUID) error {
	// Send query to database.
	if err := q.Where("user_id = ?", userID).Delete(&models.EmailVerification{}).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

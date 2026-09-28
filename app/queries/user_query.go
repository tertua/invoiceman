package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// UserQueries struct for queries from User model.
type UserQueries struct {
	*gorm.DB
}

// GetUserByID query for getting one User by given ID.
func (q *UserQueries) GetUserByID(id uuid.UUID) (models.User, error) {
	// Define User variable.
	user := models.User{}

	// Send query to database.
	if err := q.Where("id = ?", id).First(&user).Error; err != nil {
		// Return empty object and error.
		return user, notFound(err)
	}

	// Return query result.
	return user, nil
}

// GetUserByEmail query for getting one User by given Email.
func (q *UserQueries) GetUserByEmail(email string) (models.User, error) {
	// Define User variable.
	user := models.User{}

	// Send query to database.
	if err := q.Where("email = ?", email).First(&user).Error; err != nil {
		// Return empty object and error.
		return user, notFound(err)
	}

	// Return query result.
	return user, nil
}

// CreateUser query for creating a new user by given email and password hash.
func (q *UserQueries) CreateUser(u *models.User) error {
	// Send query to database.
	if err := q.Create(u).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

// CountUsers returns the number of registered users.
func (q *UserQueries) CountUsers() (int64, error) {
	var count int64
	if err := q.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ClaimFirstAdmin tries to claim the singleton first-admin row for a new
// user. Returns true only for the winner; a lost race returns (false, nil)
// once the winner's row is visible. A genuine write failure (no row
// present afterwards) is returned so the caller can fail the register.
func (q *UserQueries) ClaimFirstAdmin(userID uuid.UUID) (bool, error) {
	claim := &models.AdminClaim{ID: 1, UserID: userID, CreatedAt: time.Now()}
	if err := q.Create(claim).Error; err != nil {
		var existing models.AdminClaim
		if rerr := q.Where("id = ?", 1).First(&existing).Error; rerr == nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListUsers returns one page of users without loading any related data.
func (q *UserQueries) ListUsers(limit, offset int) ([]models.User, error) {
	users := make([]models.User, 0)
	if err := q.Order("created_at ASC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserRole changes a user's role.
func (q *UserQueries) UpdateUserRole(id uuid.UUID, role string) error {
	return q.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"updated_at": time.Now(),
		"user_role":  role,
	}).Error
}

// UpdateUserProfile query for updating user display name.
func (q *UserQueries) UpdateUserProfile(id uuid.UUID, name string) error {
	// Send query to database.
	if err := q.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"updated_at": time.Now(),
		"name":       name,
	}).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

// UpdateUserPassword query for updating user password hash.
func (q *UserQueries) UpdateUserPassword(id uuid.UUID, passwordHash string) error {
	// Send query to database.
	if err := q.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"updated_at":    time.Now(),
		"password_hash": passwordHash,
	}).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

// CreatePasswordReset query for storing a password reset token.
func (q *UserQueries) CreatePasswordReset(userID uuid.UUID, token string, expiresAt time.Time) error {
	// Send query to database.
	reset := &models.PasswordReset{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	if err := q.Create(reset).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

// GetPasswordReset query for getting a password reset token.
func (q *UserQueries) GetPasswordReset(token string) (models.PasswordReset, error) {
	// Define password reset variable.
	reset := models.PasswordReset{}

	// Send query to database.
	if err := q.Where("token = ?", token).First(&reset).Error; err != nil {
		// Return empty object and error.
		return reset, notFound(err)
	}

	// Return query result.
	return reset, nil
}

// DeletePasswordResetsByUser query for deleting all reset tokens of a user.
func (q *UserQueries) DeletePasswordResetsByUser(userID uuid.UUID) error {
	// Send query to database.
	if err := q.Where("user_id = ?", userID).Delete(&models.PasswordReset{}).Error; err != nil {
		// Return only error.
		return err
	}

	// This query returns nothing.
	return nil
}

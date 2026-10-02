package controllers

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// adminUserResponse returns the user fields exposed to an administrator.
func adminUserResponse(user models.User) adminUserRow {
	return adminUserRow{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.UserRole,
		Status:    user.UserStatus,
		CreatedAt: user.CreatedAt,
	}
}

type adminUserRow struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type auditLogRow struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

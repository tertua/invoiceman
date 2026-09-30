package queries

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// rideOrg applies the repo's rides-as fallback (same policy as CreateClient):
// an unscoped create takes the creator's user id as org_id so the row carries a
// stable scope instead of NULL. An org set by the caller rides as-is.
func rideOrg(db *gorm.DB, row any, orgID *uuid.UUID, userID uuid.UUID) error {
	if *orgID == uuid.Nil {
		*orgID = userID
	}
	return db.Create(row).Error
}

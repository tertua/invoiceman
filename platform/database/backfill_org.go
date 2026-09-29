package database

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// backfillOrganizations gives each legacy user a personal owner org and stamps org_id on their rows; idempotent, runs after AutoMigrate.
func backfillOrganizations(db *gorm.DB) error {
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		orgID, ok, err := ownerOrgFor(db, user)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := stampOrgColumns(db, user.ID, orgID); err != nil {
			return err
		}
	}
	return rebuildSettingsPK(db)
}

// ownerOrgFor resolves the user's owner membership, creating a personal org + owner membership when they have none yet.
func ownerOrgFor(db *gorm.DB, user models.User) (uuid.UUID, bool, error) {
	var memberships int64
	if err := db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Count(&memberships).Error; err != nil {
		return uuid.Nil, false, err
	}
	if memberships == 0 {
		org := models.Organization{ID: uuid.New(), Name: defaultOrgName(user)}
		membership := models.Membership{ID: uuid.New(), OrgID: org.ID, UserID: user.ID, Role: models.RoleOwner}
		// Both rows or neither: a lone org would strand the user with an owner membership pointing at nothing.
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&org).Error; err != nil {
				return err
			}
			return tx.Create(&membership).Error
		})
		if err != nil {
			return uuid.Nil, false, err
		}
		return org.ID, true, nil
	}
	var owner models.Membership
	err := db.Where("user_id = ? AND role = ?", user.ID, models.RoleOwner).Order("created_at, id").First(&owner).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, false, nil // staff-only member: their legacy rows belong to the owner's org, stamped by that user
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return owner.OrgID, true, nil
}

// defaultOrgName uses the user's name, falling back to the email local part, then a generic label.
func defaultOrgName(user models.User) string {
	if name := strings.TrimSpace(user.Name); name != "" {
		return name
	}
	if local, _, found := strings.Cut(strings.TrimSpace(user.Email), "@"); found && local != "" {
		return local
	}
	return "Organization"
}

// stampOrgColumns batch-fills org_id for the user's rows, skipping rows that already carry one; user_id is never rewritten.
func stampOrgColumns(db *gorm.DB, userID, orgID uuid.UUID) error {
	targets := []any{
		&models.Invoice{}, &models.Client{}, &models.Item{}, &models.Expense{}, &models.Payment{}, &models.AuditLog{},
	}
	for _, target := range targets {
		res := db.Model(target).
			Where("user_id = ? AND (org_id IS NULL OR org_id = ?)", userID, uuid.Nil).
			UpdateColumn("org_id", orgID)
		if res.Error != nil {
			return res.Error
		}
	}
	// Gateway projects backfill from owner_user_id (D8); projects without an owner keep a NULL org_id.
	res := db.Model(&models.GatewayProject{}).
		Where("owner_user_id = ? AND (org_id IS NULL OR org_id = ?)", userID, uuid.Nil).
		UpdateColumn("org_id", orgID)
	if res.Error != nil {
		return res.Error
	}
	return stampSettingsOrg(db, userID, orgID)
}

// stampSettingsOrg writes settings.org_id when the column exists; rows keep their own key otherwise.
func stampSettingsOrg(db *gorm.DB, userID, orgID uuid.UUID) error {
	if !db.Migrator().HasColumn(&models.Settings{}, "org_id") {
		return nil
	}
	return db.Model(&models.Settings{}).
		Where("user_id = ? AND (org_id IS NULL OR org_id = ?)", userID, uuid.Nil).
		UpdateColumn("org_id", orgID).Error
}

// rebuildSettingsPK is idempotent: a settings table whose org_id is already the primary key (fresh schema) or lacks the column is left alone.
// GORM cannot add a PK to an existing table, so the v16 backfill rebuilds settings once.
func rebuildSettingsPK(db *gorm.DB) error {
	cols, err := db.Migrator().ColumnTypes("settings")
	if err != nil {
		return err
	}
	orgIDSeen, orgIDIsPK := false, false
	for _, col := range cols {
		if col.Name() != "org_id" {
			continue
		}
		orgIDSeen = true
		pk, ok := col.PrimaryKey()
		orgIDIsPK = ok && pk // driver that cannot report PK: rebuild from the model definition
	}
	if !orgIDSeen || orgIDIsPK {
		return nil
	}
	var rows []models.Settings
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		if rows[i].OrgID == uuid.Nil {
			rows[i].OrgID = rows[i].UserID // unstamped legacy row: keep the old user_id key as its org
		}
	}
	if err := db.Migrator().DropTable(&models.Settings{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.Settings{}); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return db.CreateInBatches(&rows, 100).Error
}

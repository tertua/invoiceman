package queries

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// rideTestDB opens a private in-memory SQLite with the item table, mirroring the middleware harness.
func rideTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Item{}))
	return db
}

// TestCreateItemRidesAsUserWhenOrgUnset proves the tenant fallback: a catalog item created without an org is stamped with the creator's user id instead of a NULL org that would leak across tenants.
func TestCreateItemRidesAsUserWhenOrgUnset(t *testing.T) {
	db := rideTestDB(t)
	userID := uuid.New()
	item := models.Item{ID: uuid.New(), UserID: userID, Name: "Service"}
	require.NoError(t, (&ItemQueries{DB: db}).CreateItem(&item))

	var stored models.Item
	require.NoError(t, db.Where("id = ?", item.ID).First(&stored).Error)
	assert.Equal(t, userID, stored.OrgID)
}

// TestCreateItemKeepsCallerOrg proves a scoped create is untouched: an org set by the caller (a joined org, not the personal one) rides as-is.
func TestCreateItemKeepsCallerOrg(t *testing.T) {
	db := rideTestDB(t)
	userID, orgID := uuid.New(), uuid.New()
	item := models.Item{ID: uuid.New(), UserID: userID, OrgID: orgID, Name: "Service"}
	require.NoError(t, (&ItemQueries{DB: db}).CreateItem(&item))

	var stored models.Item
	require.NoError(t, db.Where("id = ?", item.ID).First(&stored).Error)
	assert.Equal(t, orgID, stored.OrgID)
}

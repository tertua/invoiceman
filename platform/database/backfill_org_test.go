package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// TestBackfillStampsOwnerlessGatewayProject pins the tenant stamp for a legacy
// relay project registered before orgs existed and never given an owner.
//
// The project survives the migration with org_id = NULL, and the intent path
// then reads GetSettings(project.OrgID) — so a project without a tenant
// silently falls through to the nil org. Every project that outlives the
// backfill must carry an org.
func TestBackfillStampsOwnerlessGatewayProject(t *testing.T) {
	db := probeDB(t)
	probeMigrate(t, db)

	owner := models.User{ID: uuid.New(), Name: "Relay Owner", Email: "relay-owner@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("expected probe user, got: %v", err)
	}

	// Legacy shape: a project with no owner_user_id and no org_id at all.
	legacy := models.GatewayProject{
		Slug: "legacy-relay", Name: "Legacy Relay", WebhookURL: "https://relay.example.com/hook",
		APIKeyHash: "legacy-hash", WebhookSecret: "secret", DefaultGateway: "midtrans", IsActive: true,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("expected legacy project, got: %v", err)
	}

	if err := backfillOrganizations(db); err != nil {
		t.Fatalf("expected backfill to succeed, got: %v", err)
	}

	var project models.GatewayProject
	if err := db.Where("slug = ?", legacy.Slug).First(&project).Error; err != nil {
		t.Fatalf("expected project to survive the backfill, got: %v", err)
	}
	if project.OrgID == uuid.Nil {
		t.Fatalf("expected the ownerless project to be stamped with an org, got nil")
	}
	// The stamp must point at a real org so settings and invoices resolve.
	var org models.Organization
	if err := db.Where("id = ?", project.OrgID).First(&org).Error; err != nil {
		t.Fatalf("expected the stamped org to exist, got: %v", err)
	}

	// A second run must reuse that org for the project, not mint another one.
	if err := backfillOrganizations(db); err != nil {
		t.Fatalf("expected the second backfill to succeed, got: %v", err)
	}
	var after models.GatewayProject
	if err := db.Where("slug = ?", legacy.Slug).First(&after).Error; err != nil {
		t.Fatalf("expected project after the second backfill, got: %v", err)
	}
	if after.OrgID != project.OrgID {
		t.Errorf("expected the project to keep org %s, got %s", project.OrgID, after.OrgID)
	}
	var systemOrgs int64
	if err := db.Model(&models.Organization{}).Where("name = ?", systemOrgName).Count(&systemOrgs).Error; err != nil {
		t.Fatalf("expected to count system orgs, got: %v", err)
	}
	if systemOrgs != 1 {
		t.Errorf("expected one system org after two backfills, got %d", systemOrgs)
	}
}

// TestBackfillStampsGatewayProjectFromOwner proves the common case still holds:
// an owned project is stamped with the owner's org, not a second one.
func TestBackfillStampsGatewayProjectFromOwner(t *testing.T) {
	db := probeDB(t)
	probeMigrate(t, db)

	owner := models.User{ID: uuid.New(), Name: "Owned Relay", Email: "owned-relay@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("expected probe user, got: %v", err)
	}
	project := models.GatewayProject{
		Slug: "owned-relay", Name: "Owned Relay", WebhookURL: "https://relay.example.com/hook",
		APIKeyHash: "owned-hash", WebhookSecret: "secret", DefaultGateway: "midtrans", IsActive: true,
		GatewayProjectOwner: models.GatewayProjectOwner{OwnerUserID: &owner.ID},
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("expected owned project, got: %v", err)
	}

	if err := backfillOrganizations(db); err != nil {
		t.Fatalf("expected backfill to succeed, got: %v", err)
	}

	var stamped models.GatewayProject
	if err := db.Where("slug = ?", project.Slug).First(&stamped).Error; err != nil {
		t.Fatalf("expected project to survive, got: %v", err)
	}
	var membership models.Membership
	if err := db.Where("user_id = ?", owner.ID).First(&membership).Error; err != nil {
		t.Fatalf("expected a personal membership for the owner, got: %v", err)
	}
	if stamped.OrgID != membership.OrgID {
		t.Errorf("expected project org %s to be the owner's org %s", stamped.OrgID, membership.OrgID)
	}
}

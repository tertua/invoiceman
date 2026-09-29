package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type orgResult struct {
	orgID   uuid.UUID
	orgErr  error
	role    string
	roleErr error
	absent  uuid.UUID
	actor   uuid.UUID
}

func runOrgRoute(t *testing.T, path string, setup func(c fiber.Ctx)) orgResult {
	t.Helper()
	var got orgResult
	app := fiber.New()
	app.Get(path, func(c fiber.Ctx) error {
		setup(c)
		got.orgID, got.orgErr = CurrentOrgID(c)
		got.role, got.roleErr = CurrentOrgRole(c)
		got.absent = OrgIDFromLocals(c)
		got.actor = CurrentActorID(c)
		return c.SendStatus(http.StatusOK)
	})
	resp, err := app.Test(httptest.NewRequest("GET", path, http.NoBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return got
}

func TestOrgLocalsPresent(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	got := runOrgRoute(t, "/with", func(c fiber.Ctx) {
		c.Locals(SessionUserIDKey, userID)
		c.Locals(SessionOrgIDKey, orgID)
		c.Locals(SessionOrgRoleKey, "owner")
	})
	if got.orgErr != nil || got.orgID != orgID {
		t.Errorf("CurrentOrgID = %v, %v; want %v, nil", got.orgID, got.orgErr, orgID)
	}
	if got.roleErr != nil || got.role != "owner" {
		t.Errorf("CurrentOrgRole = %q, %v; want owner, nil", got.role, got.roleErr)
	}
	if got.absent != orgID {
		t.Errorf("OrgIDFromLocals = %v; want %v", got.absent, orgID)
	}
	if got.actor != userID {
		t.Errorf("CurrentActorID = %v; want %v", got.actor, userID)
	}
}

func TestOrgLocalsMissing(t *testing.T) {
	got := runOrgRoute(t, "/without", func(c fiber.Ctx) {})
	if !errors.Is(got.orgErr, ErrOrgNotMember) || got.orgID != uuid.Nil {
		t.Errorf("CurrentOrgID = %v, %v; want uuid.Nil, ErrOrgNotMember", got.orgID, got.orgErr)
	}
	if !errors.Is(got.roleErr, ErrOrgNotMember) || got.role != "" {
		t.Errorf("CurrentOrgRole = %q, %v; want empty, ErrOrgNotMember", got.role, got.roleErr)
	}
	if got.absent != uuid.Nil {
		t.Errorf("OrgIDFromLocals = %v; want uuid.Nil", got.absent)
	}
	if got.actor != uuid.Nil {
		t.Errorf("CurrentActorID = %v; want uuid.Nil", got.actor)
	}
}

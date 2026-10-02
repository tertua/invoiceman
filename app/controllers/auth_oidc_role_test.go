package controllers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tertua/tupay/pkg/repository"
)

// TestOIDCRoleFromClaims covers the supported claim shapes and fail-closed
// behaviour of the role extractor (no DB, no HTTP).
func TestOIDCRoleFromClaims(t *testing.T) {
	raw := func(s string) json.RawMessage { return json.RawMessage(s) }

	tests := []struct {
		name      string
		payload   string
		path      string
		adminRole string
		wantRole  string
		wantAdmin bool
	}{
		{"top-level string admin", `{"roles":"admin"}`, "roles", "admin", repository.AdminRoleName, true},
		{"top-level array admin", `{"roles":["admin","x"]}`, "roles", "admin", repository.AdminRoleName, true},
		{"case-insensitive match", `{"roles":["ADMIN"]}`, "roles", "admin", repository.AdminRoleName, true},
		{"non-admin value", `{"roles":["viewer"]}`, "roles", "admin", repository.UserRoleName, false},
		{"plain user value", `{"roles":"user"}`, "roles", "admin", repository.UserRoleName, false},
		{"nested admin", `{"realm_access":{"roles":["admin"]}}`, "realm_access.roles", "admin", repository.AdminRoleName, true},
		{"nested absent path", `{}`, "realm_access.roles", "admin", repository.UserRoleName, false},
		{"intermediate not object", `{"realm_access":"x"}`, "realm_access.roles", "admin", repository.UserRoleName, false},
		{"final node object", `{"roles":{"a":"b"}}`, "roles", "admin", repository.UserRoleName, false},
		{"final node number", `{"roles":7}`, "roles", "admin", repository.UserRoleName, false},
		{"final node bool", `{"roles":true}`, "roles", "admin", repository.UserRoleName, false},
		{"mixed array keeps strings", `{"roles":[1,"admin"]}`, "roles", "admin", repository.AdminRoleName, true},
		{"array without strings", `{"roles":[1,2]}`, "roles", "admin", repository.UserRoleName, false},
		{"custom admin value", `{"roles":["superuser"]}`, "roles", "superuser", repository.AdminRoleName, true},
		{"custom admin value missing", `{"roles":["admin"]}`, "roles", "superuser", repository.UserRoleName, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			role, isAdmin := oidcRoleFromClaims(raw(tc.payload), tc.path, tc.adminRole)
			assert.Equal(t, tc.wantRole, role)
			assert.Equal(t, tc.wantAdmin, isAdmin)
		})
	}

	t.Run("malformed JSON fails closed", func(t *testing.T) {
		role, isAdmin := oidcRoleFromClaims(raw("{"), "roles", "admin")
		assert.Equal(t, repository.UserRoleName, role)
		assert.False(t, isAdmin)
	})
	t.Run("empty raw fails closed", func(t *testing.T) {
		role, isAdmin := oidcRoleFromClaims(nil, "roles", "admin")
		assert.Equal(t, repository.UserRoleName, role)
		assert.False(t, isAdmin)
	})
	t.Run("empty path is inert", func(t *testing.T) {
		role, isAdmin := oidcRoleFromClaims(raw(`{"roles":["admin"]}`), "", "admin")
		assert.Empty(t, role)
		assert.False(t, isAdmin)
	})
}

// TestExtractRawClaim checks the re-marshal helper walks nested paths and
// returns nil for a miss.
func TestExtractRawClaim(t *testing.T) {
	all := map[string]any{
		"roles":        []any{"admin"},
		"realm_access": map[string]any{"roles": []any{"admin", "user"}},
	}

	t.Run("top-level node", func(t *testing.T) {
		got := extractRawClaim(all, "roles")
		assert.JSONEq(t, `["admin"]`, string(got))
	})
	t.Run("nested node", func(t *testing.T) {
		got := extractRawClaim(all, "realm_access.roles")
		assert.JSONEq(t, `["admin","user"]`, string(got))
	})
	t.Run("absent path", func(t *testing.T) {
		assert.Nil(t, extractRawClaim(all, "realm_access.missing"))
	})
	t.Run("empty path", func(t *testing.T) {
		assert.Nil(t, extractRawClaim(all, ""))
	})
	t.Run("nil map", func(t *testing.T) {
		assert.Nil(t, extractRawClaim(nil, "roles"))
	})
}

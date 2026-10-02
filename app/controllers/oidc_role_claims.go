package controllers

import (
	"encoding/json"
	"strings"

	"github.com/tertua/tupay/pkg/repository"
)

// oidcRoleFromClaims extracts the role list at path (dot-separated) from the
// verified ID-token claims and reports whether adminRole grants platform admin.
// It fails closed: a missing path, a type mismatch or malformed JSON returns
// ("user", false) and never a stale admin.
//
// Supported shapes at the final segment: []string, string, []any. Intermediate
// segments must be JSON objects (map[string]any). Matching is case-insensitive.
func oidcRoleFromClaims(raw json.RawMessage, path, adminRole string) (role string, isAdmin bool) {
	if path == "" {
		return "", false
	}
	if len(raw) == 0 {
		return repository.UserRoleName, false
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return repository.UserRoleName, false
	}

	cur := any(root)
	segs := strings.Split(path, ".")
	for i := 0; i < len(segs)-1; i++ {
		m, ok := cur.(map[string]any)
		if !ok {
			return repository.UserRoleName, false
		}
		cur, ok = m[segs[i]]
		if !ok {
			return repository.UserRoleName, false
		}
	}
	m, ok := cur.(map[string]any)
	if !ok {
		return repository.UserRoleName, false
	}
	node, ok := m[segs[len(segs)-1]]
	if !ok {
		return repository.UserRoleName, false
	}

	for _, v := range roleValues(node) {
		if strings.EqualFold(v, adminRole) {
			return repository.AdminRoleName, true
		}
	}
	return repository.UserRoleName, false
}

// extractRawClaim walks the already-decoded claim map for the configured path
// and re-marshals the node so the sync path shares one representation. A missing
// path yields nil (which the role parser treats as "no role").
func extractRawClaim(all map[string]any, path string) json.RawMessage {
	if path == "" || all == nil {
		return nil
	}
	cur := any(all)
	segs := strings.Split(path, ".")
	for i := 0; i < len(segs)-1; i++ {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[segs[i]]
		if !ok {
			return nil
		}
	}
	m, ok := cur.(map[string]any)
	if !ok {
		return nil
	}
	node, ok := m[segs[len(segs)-1]]
	if !ok {
		return nil
	}
	raw, err := json.Marshal(node)
	if err != nil {
		return nil
	}
	return raw
}

// roleValues normalizes the final claim node into a list of strings; unknown
// types yield nil (treated as "no role").
func roleValues(node any) []string {
	switch v := node.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

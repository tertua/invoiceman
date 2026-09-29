package cache

import (
	"strings"
	"time"

	"github.com/tertua/tupay/pkg/configs"
)

// RefreshTTL returns the refresh token lifetime from the central config.
func RefreshTTL() time.Duration {
	return time.Hour * time.Duration(configs.Get().JWT.RefreshHours)
}

// EncodeSessionValue packs sid, refresh, CSRF and active org with "\n"; tokens are hex(+dots)/UUIDs, never multiline.
func EncodeSessionValue(sid, refresh, csrf, activeOrgID string) string {
	return sid + "\n" + refresh + "\n" + csrf + "\n" + activeOrgID
}

// DecodeSessionValue splits a stored session value into its parts; legacy entries predate fields (bare refresh → empty sid/csrf, two-part → empty csrf, three-part → empty active org) and fields beyond active org are ignored.
func DecodeSessionValue(value string) (sid, refresh, csrf, activeOrgID string, ok bool) {
	parts := strings.Split(value, "\n")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", "", "", false
		}
		return "", parts[0], "", "", true
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", "", "", false
		}
		return parts[0], parts[1], "", "", true
	case 3:
		if parts[0] == "" || parts[1] == "" {
			return "", "", "", "", false
		}
		return parts[0], parts[1], parts[2], "", true
	default:
		if parts[0] == "" || parts[1] == "" {
			return "", "", "", "", false
		}
		return parts[0], parts[1], parts[2], parts[3], true
	}
}

package cache

import (
	"context"
	"errors"
)

// SessionValue is the decoded four-field session payload (sid, refresh, CSRF, active org).
type SessionValue struct {
	SID         string
	Refresh     string
	CSRF        string
	ActiveOrgID string
}

// ReadSessionValue loads the stored session under key and decodes its parts; ok is false when the entry is absent or malformed.
func ReadSessionValue(ctx context.Context, key string) (SessionValue, bool, error) {
	store, err := Sessions()
	if err != nil {
		return SessionValue{}, false, err
	}
	raw, err := store.Get(ctx, key)
	if errors.Is(err, ErrSessionNotFound) {
		return SessionValue{}, false, nil
	}
	if err != nil {
		return SessionValue{}, false, err
	}
	sid, refresh, csrf, org, ok := DecodeSessionValue(raw)
	return SessionValue{SID: sid, Refresh: refresh, CSRF: csrf, ActiveOrgID: org}, ok, nil
}

// WriteSessionValue encodes v as a four-field value and stores it under key for RefreshTTL.
func WriteSessionValue(ctx context.Context, key string, v SessionValue) error {
	store, err := Sessions()
	if err != nil {
		return err
	}
	return store.Set(ctx, key, EncodeSessionValue(v.SID, v.Refresh, v.CSRF, v.ActiveOrgID), RefreshTTL())
}

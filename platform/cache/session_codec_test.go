package cache

import (
	"strings"
	"testing"
)

func TestSessionCodecRoundTrip(t *testing.T) {
	value := EncodeSessionValue("sid-1", "refresh.token.1", "csrfhex", "org-1")
	if value != "sid-1\nrefresh.token.1\ncsrfhex\norg-1" {
		t.Fatalf("unexpected encoded value %q", value)
	}
	sid, refresh, csrf, org, ok := DecodeSessionValue(value)
	if !ok || sid != "sid-1" || refresh != "refresh.token.1" || csrf != "csrfhex" || org != "org-1" {
		t.Fatalf("roundtrip mismatch: %q %q %q %q %v", sid, refresh, csrf, org, ok)
	}
	if RefreshTTL() <= 0 {
		t.Fatalf("expected positive refresh TTL, got %v", RefreshTTL())
	}
}

// TestSessionCodecEncodeAlwaysFour asserts encode always writes four fields, empty active org included.
func TestSessionCodecEncodeAlwaysFour(t *testing.T) {
	value := EncodeSessionValue("sid-1", "refresh.token.1", "csrfhex", "")
	if got := len(strings.Split(value, "\n")); got != 4 {
		t.Fatalf("encode must always write 4 parts, got %d", got)
	}
	sid, refresh, csrf, org, ok := DecodeSessionValue(value)
	if !ok || sid != "sid-1" || refresh != "refresh.token.1" || csrf != "csrfhex" || org != "" {
		t.Fatalf("decode of empty-org value mismatch: %q %q %q %q %v", sid, refresh, csrf, org, ok)
	}
}

func TestSessionCodecLegacyFormats(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantSid string
		wantRef string
		wantCSF string
		wantOrg string
		wantOK  bool
	}{
		{"pre-sid single part", "refresh.token.1", "", "refresh.token.1", "", "", true},
		{"two-part no csrf", "sid-1\nrefresh.token.1", "sid-1", "refresh.token.1", "", "", true},
		{"three-part no org", "sid-1\nrefresh.token.1\ncsrfhex", "sid-1", "refresh.token.1", "csrfhex", "", true},
		{"empty value", "", "", "", "", "", false},
		{"empty sid", "\nrefresh.token.1", "", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sid, refresh, csrf, org, ok := DecodeSessionValue(tt.value)
			if ok != tt.wantOK || sid != tt.wantSid || refresh != tt.wantRef || csrf != tt.wantCSF || org != tt.wantOrg {
				t.Fatalf("got %q %q %q %q %v, want %q %q %q %q %v", sid, refresh, csrf, org, ok, tt.wantSid, tt.wantRef, tt.wantCSF, tt.wantOrg, tt.wantOK)
			}
		})
	}
}

// TestSessionCodecTrailingFieldsIgnored asserts fields beyond active org never leak into the decode.
func TestSessionCodecTrailingFieldsIgnored(t *testing.T) {
	sid, refresh, csrf, org, ok := DecodeSessionValue("sid-1\nrefresh.token.1\ncsrfhex\norg-1\nextra")
	if !ok || sid != "sid-1" || refresh != "refresh.token.1" || csrf != "csrfhex" || org != "org-1" {
		t.Fatalf("4-part decode mismatch: %q %q %q %q %v", sid, refresh, csrf, org, ok)
	}
}

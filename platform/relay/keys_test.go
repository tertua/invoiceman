package relay

import (
	"strings"
	"testing"
)

func TestAPIKeyRoundTrip(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil || !strings.HasPrefix(key, "inv_live_") {
		t.Fatalf("bad api key: %v %q", err, key)
	}
	hash := HashKey(key)
	if len(hash) != 64 || hash == key {
		t.Fatal("api key must be stored as SHA256 hash")
	}
	if ExtractKey("Bearer "+key, "") != key {
		t.Fatal("bearer extraction failed")
	}
	if ExtractKey("", key) != key {
		t.Fatal("x-api-key fallback failed")
	}
}

func TestSignVerify(t *testing.T) {
	body := []byte(`{"order_id":"EXT-one-api-topup_1-x"}`)
	secret, _ := GenerateSecret()
	sig := SignPayload(body, secret)
	if !VerifySignature(body, secret, sig) {
		t.Fatal("valid relay signature rejected")
	}
	if VerifySignature(body, secret, "bad") {
		t.Fatal("invalid relay signature accepted")
	}
	if VerifySignature(body, "other", sig) {
		t.Fatal("wrong secret accepted")
	}
}

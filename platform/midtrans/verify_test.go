package midtrans

import (
	"crypto/sha512"
	"encoding/hex"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	n := &Notification{OrderID: "EXT-one-api-topup_abc-x1", StatusCode: "200", GrossAmount: "32000.00"}
	key := "test-server-key"
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + key))
	n.SignatureKey = hex.EncodeToString(sum[:])
	if !VerifySignature(n, key) {
		t.Fatal("valid signature rejected")
	}
	n.SignatureKey = "bad"
	if VerifySignature(n, key) {
		t.Fatal("invalid signature accepted")
	}
}

func TestMapStatus(t *testing.T) {
	cases := map[string]string{
		"settlement": "success", "capture": "success",
		"pending": "pending", "expire": "expired",
		"deny": "failed", "cancel": "failed", "failure": "failed",
		"refund": "refunded", "partial_refund": "partially_refunded",
		"unknown-xyz": "pending",
	}
	for in, want := range cases {
		if got := MapStatus(in); got != want {
			t.Fatalf("MapStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

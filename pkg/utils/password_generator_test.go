package utils

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestGeneratePasswordRoundTrip pins hashing: the produced hash verifies its own
// password and rejects any other.
func TestGeneratePasswordRoundTrip(t *testing.T) {
	hash := GeneratePassword("secret123")
	if !ComparePasswords(hash, "secret123") {
		t.Fatal("expected the hash to verify its own password")
	}
	if ComparePasswords(hash, "wrong123") {
		t.Fatal("expected a wrong password to be rejected")
	}
}

// TestGeneratePasswordCost pins the work factor at bcrypt.DefaultCost (10) so a
// regression back to MinCost (4) fails loudly.
func TestGeneratePasswordCost(t *testing.T) {
	hash := GeneratePassword("secret123")
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("expected a readable cost, got: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Fatalf("expected cost %d, got %d", bcrypt.DefaultCost, cost)
	}
}

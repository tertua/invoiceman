package utils

import (
	"testing"

	"github.com/google/uuid"
)

// The custom "uuid" rule must accept parseable IDs in both shapes used by
// models: uuid.UUID-typed fields and client-supplied strings. It previously
// returned the inverted result for strings (rejecting every valid ID, which
// broke payment reminders) while accidentally passing typed fields whose
// reflect String() never parses.
func TestUUIDValidation(t *testing.T) {
	id := uuid.New()

	typed := struct {
		ID uuid.UUID `validate:"required,uuid"`
	}{ID: id}
	if err := NewValidator().Struct(typed); err != nil {
		t.Fatalf("typed UUID field rejected: %v", err)
	}

	str := struct {
		InvoiceID string `validate:"required,uuid"`
	}{InvoiceID: id.String()}
	if err := NewValidator().Struct(str); err != nil {
		t.Fatalf("valid UUID string rejected: %v", err)
	}

	bad := struct {
		InvoiceID string `validate:"required,uuid"`
	}{InvoiceID: "not-a-uuid"}
	if err := NewValidator().Struct(bad); err == nil {
		t.Fatal("invalid UUID string accepted")
	}
}

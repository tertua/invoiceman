package controllers

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func marshalKeys(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	blob, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(blob, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

// TestInvoiceDetailJSONShape pins the internal detail key set and the
// external_id omitempty contract the gateway route relies on.
func TestInvoiceDetailJSONShape(t *testing.T) {
	detail := invoiceDetailResponse{
		ID:              uuid.New(),
		InvoiceNumber:   "INV-1",
		Status:          "sent",
		EffectiveStatus: "sent",
		ClientID:        &uuid.Nil,
		Items:           []invoiceItemResponse{},
		Payments:        []invoicePaymentResponse{},
	}

	keys := marshalKeys(t, detail)
	for _, want := range []string{
		"id", "invoice_number", "status", "effective_status", "payment_link",
		"client_id", "client_name", "client_company", "client_email",
		"issue_date", "due_date", "currency", "subtotal", "discount",
		"tax_rate", "tax_amount", "total", "notes", "terms",
		"payment_method", "items", "payments", "paid_amount", "balance",
	} {
		if _, ok := keys[want]; !ok {
			t.Errorf("internal detail missing key %q", want)
		}
	}
	if _, ok := keys["external_id"]; ok {
		t.Error("external_id must be absent when unset (non-gateway callers)")
	}
	if string(keys["payment_link"]) != "null" {
		t.Errorf("payment_link without link = %s, want null", keys["payment_link"])
	}
	if len(keys) != 24 {
		t.Errorf("internal detail has %d keys, want 24", len(keys))
	}

	external := "order-1"
	detail.ExternalID = &external
	if got := marshalKeys(t, detail)["external_id"]; string(got) != `"order-1"` {
		t.Errorf("external_id = %s, want \"order-1\"", got)
	}
}

// TestPublicInvoiceDetailJSONShape pins the payer-facing payload: the
// embedded detail's private fields disappear and the payment rows carry only
// the four public keys.
func TestPublicInvoiceDetailJSONShape(t *testing.T) {
	detail := invoiceDetailResponse{
		ID:              uuid.New(),
		InvoiceNumber:   "INV-1",
		Status:          "sent",
		EffectiveStatus: "sent",
		ClientID:        &uuid.Nil,
		ClientEmail:     "billing@example.com",
		PaymentLink:     &paymentLinkResponse{Token: "tok", URL: "/pay/tok"},
		Items:           []invoiceItemResponse{},
		Payments: []invoicePaymentResponse{{
			ID: uuid.New(), TxnID: "txn-1", CanVoid: true,
		}},
	}
	pub := publicInvoiceDetailResponse{
		invoiceDetailResponse: detail,
		Payments: []publicPaymentResponse{{
			ID: uuid.New(), Method: "qris",
		}},
	}

	keys := marshalKeys(t, pub)
	for _, hidden := range []string{"client_id", "client_email", "payment_link"} {
		if _, ok := keys[hidden]; ok {
			t.Errorf("public detail must not expose %q", hidden)
		}
	}
	for _, want := range []string{"id", "invoice_number", "effective_status", "client_name", "payments", "balance"} {
		if _, ok := keys[want]; !ok {
			t.Errorf("public detail missing key %q", want)
		}
	}
	if len(keys) != 21 {
		t.Errorf("public detail has %d keys, want 21 (internal 24 minus 3 hidden)", len(keys))
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(keys["payments"], &rows); err != nil {
		t.Fatalf("payments: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("payments rows = %d, want 1", len(rows))
	}
	if len(rows[0]) != 4 {
		t.Errorf("public payment row has %d keys, want 4 (id, amount, paid_on, method)", len(rows[0]))
	}
	for _, public := range []string{"id", "amount", "paid_on", "method"} {
		if _, ok := rows[0][public]; !ok {
			t.Errorf("public payment row missing %q", public)
		}
	}
	if _, ok := rows[0]["txn_id"]; ok {
		t.Error("public payment row must not leak txn_id")
	}
	if _, ok := rows[0]["can_void"]; ok {
		t.Error("public payment row must not leak can_void")
	}
}

package models

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tertua/invoiceman/platform/gateway"
)

// The validator enum on IntentInput.PaymentMethod must stay in lockstep with
// the provider-neutral method ids; a drifted list would reject valid methods
// (or accept ones routing cannot serve).
func TestIntentPaymentMethodEnumMatchesGatewayIDs(t *testing.T) {
	field, ok := reflect.TypeOf(IntentInput{}).FieldByName("PaymentMethod")
	if !ok {
		t.Fatal("IntentInput.PaymentMethod field missing")
	}
	var oneof []string
	for _, part := range strings.Split(field.Tag.Get("validate"), ",") {
		if strings.HasPrefix(part, "oneof=") {
			oneof = strings.Fields(strings.TrimPrefix(part, "oneof="))
		}
	}
	if !reflect.DeepEqual(oneof, gateway.IDs()) {
		t.Fatalf("payment_method oneof = %v, want gateway.IDs() = %v", oneof, gateway.IDs())
	}
}

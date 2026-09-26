package models

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// The invoice payment_method validator enum must stay in lockstep with the
// SPA's shared list (web/src/lib/paymentMethods.js): a drifted list would
// either reject choices the UI offers or show options the API rejects.
func TestInvoicePaymentMethodEnumMatchesFrontend(t *testing.T) {
	oneof, ok := oneOfValues(fieldValidateTag(t, InvoiceInput{}, "PaymentMethod"))
	if !ok {
		t.Fatal("InvoiceInput.PaymentMethod has no oneof validate tag")
	}

	path := filepath.Join("..", "..", "web", "src", "lib", "paymentMethods.js")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read frontend enum: %v", err)
	}
	frontend := jsStringArray(t, src, "PAYMENT_METHODS")

	if !reflect.DeepEqual(oneof, frontend) {
		t.Fatalf("InvoiceInput.PaymentMethod oneof = %v, want web PAYMENT_METHODS = %v", oneof, frontend)
	}

	// The bare Invoice column repeats the tag for swagger; keep both Go tags
	// from drifting apart too.
	invoiceOneof, ok := oneOfValues(fieldValidateTag(t, Invoice{}, "PaymentMethod"))
	if !ok || !reflect.DeepEqual(invoiceOneof, oneof) {
		t.Fatalf("Invoice.PaymentMethod oneof = %v, want %v", invoiceOneof, oneof)
	}
}

// fieldValidateTag returns the validate tag of a struct field.
func fieldValidateTag(t *testing.T, v any, name string) string {
	t.Helper()
	field, ok := reflect.TypeOf(v).FieldByName(name)
	if !ok {
		t.Fatalf("%T.%s field missing", v, name)
	}
	return field.Tag.Get("validate")
}

// oneOfValues extracts the values of a validator `oneof=` directive,
// respecting single/double quotes so values may contain spaces.
func oneOfValues(tag string) ([]string, bool) {
	for _, part := range strings.Split(tag, ",") {
		if !strings.HasPrefix(part, "oneof=") {
			continue
		}
		var out []string
		var cur strings.Builder
		var quote rune
		for _, r := range strings.TrimPrefix(part, "oneof=") {
			switch {
			case quote != 0:
				if r == quote {
					out = append(out, cur.String())
					cur.Reset()
					quote = 0
				} else {
					cur.WriteRune(r)
				}
			case r == '\'' || r == '"':
				quote = r
			case r == ' ':
				if cur.Len() > 0 {
					out = append(out, cur.String())
					cur.Reset()
				}
			default:
				cur.WriteRune(r)
			}
		}
		if cur.Len() > 0 {
			out = append(out, cur.String())
		}
		return out, true
	}
	return nil, false
}

// jsStringArray parses a top-level `name = ["a", "b"]` array of strings.
func jsStringArray(t *testing.T, src []byte, name string) []string {
	t.Helper()
	array := regexp.MustCompile(`(?s)` + name + `\s*=\s*\[(.*?)\]`).FindSubmatch(src)
	if array == nil {
		t.Fatalf("could not find %s array in frontend module", name)
	}
	matches := regexp.MustCompile(`"([^"]*)"`).FindAllSubmatch(array[1], -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, string(m[1]))
	}
	if len(out) == 0 {
		t.Fatalf("%s array parsed empty", name)
	}
	return out
}

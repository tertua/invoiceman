package midtrans

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/platform/gateway"
)

// Notification is the subset of the Midtrans payment notification used here.
type Notification struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	TransactionStatus string `json:"transaction_status"`
	PaymentType       string `json:"payment_type"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
}

// ParseNotification decodes a Midtrans notification body.
func ParseNotification(raw []byte) (*Notification, error) {
	n := &Notification{}
	if err := json.Unmarshal(raw, n); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON", gateway.ErrInvalidPayload)
	}
	if strings.TrimSpace(n.OrderID) == "" {
		return nil, fmt.Errorf("%w: missing order_id", gateway.ErrInvalidPayload)
	}
	return n, nil
}

// VerifySignature checks signature_key == SHA512(order_id+status_code+gross_amount+serverKey).
func VerifySignature(n *Notification, serverKey string) bool {
	if n == nil || n.SignatureKey == "" || serverKey == "" {
		return false
	}
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + serverKey))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(n.SignatureKey), []byte(expected)) == 1
}

// GrossAmountValue parses the gross_amount string as a fixed-point decimal.
func (n *Notification) GrossAmountValue() decimal.Decimal {
	v, err := decimal.NewFromString(strings.TrimSpace(n.GrossAmount))
	if err != nil {
		return decimal.Zero
	}
	return v
}

package midtrans

import (
	"strings"

	"github.com/tertua/invoiceman/platform/gateway"
)

var statusMap = map[string]string{
	"settlement": gateway.StatusSuccess, "capture": gateway.StatusSuccess,
	"expire": gateway.StatusExpired, "deny": gateway.StatusFailed, "cancel": gateway.StatusFailed, "failure": gateway.StatusFailed,
	"refund": gateway.StatusRefunded, "partial_refund": gateway.StatusPartialRefunded,
}

// MapStatus converts a Midtrans transaction_status to a relay status.
func MapStatus(transactionStatus string) string {
	if status, ok := statusMap[strings.ToLower(strings.TrimSpace(transactionStatus))]; ok {
		return status
	}
	return gateway.StatusPending
}

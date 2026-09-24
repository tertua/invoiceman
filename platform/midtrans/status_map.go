package midtrans

import "strings"

// MapStatus converts a Midtrans transaction_status to a relay status.
func MapStatus(transactionStatus string) string {
	switch strings.ToLower(strings.TrimSpace(transactionStatus)) {
	case "settlement", "capture":
		return "success"
	case "expire":
		return "expired"
	case "deny", "cancel", "failure":
		return "failed"
	case "refund":
		return "refunded"
	case "partial_refund":
		return "partially_refunded"
	default:
		return "pending"
	}
}

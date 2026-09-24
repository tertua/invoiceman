package controllers

import "strings"

func normalizedPaymentMethod(method string) string {
	return strings.ToLower(strings.TrimSpace(method))
}

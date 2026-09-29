package controllers

import (
	"net/netip"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// invoiceNotifData builds the data block shared by invoice.* events; orgID scopes both lookups.
func invoiceNotifData(db *database.Queries, orgID, invoiceID uuid.UUID) fiber.Map {
	out := fiber.Map{"invoice_id": invoiceID.String()}
	invoice, err := db.GetInvoice(orgID, invoiceID)
	if err != nil {
		return out
	}
	out["invoice_number"] = invoice.InvoiceNumber
	out["status"] = invoice.Status
	out["issue_date"] = utils.FormatDate(invoice.IssueDate)
	out["due_date"] = utils.FormatDate(invoice.DueDate)
	out["currency"] = invoice.Currency
	out["total"] = invoice.Total
	if invoice.ClientID != nil {
		if client, err := db.GetClient(orgID, *invoice.ClientID); err == nil {
			out["client_name"] = client.Name
			out["client_company"] = client.Company
			out["client_email"] = client.Email
			out["client_phone"] = client.Phone
		}
	}
	return out
}

// paymentNotifData builds the data block for payment.created events.
func paymentNotifData(db *database.Queries, payment models.Payment) fiber.Map {
	out := fiber.Map{
		"payment_id": payment.ID.String(),
		"invoice_id": payment.InvoiceID.String(),
		"amount":     payment.Amount,
		"method":     payment.Method,
		"paid_on":    utils.FormatDate(payment.PaidOn),
	}
	for k, v := range invoiceNotifData(db, payment.OrgID, payment.InvoiceID) {
		if _, exists := out[k]; !exists {
			out[k] = v
		}
	}
	return out
}

// validateEndpointURL rejects non-http(s) targets and SSRF-prone hosts
// (loopback / private / link-local) when running in prod stage.
func validateEndpointURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Hostname() == "" {
		return false
	}
	if strings.EqualFold(configs.Get().Stage, "prod") {
		if isPrivateHost(parsed.Hostname()) {
			return false
		}
	}
	return true
}

// isPrivateHost reports loopback, private, link-local or unspecified IPs,
// plus "localhost" itself. Hostnames that do not parse as IPs are treated
// as public (DNS is resolved by the forwarder's HTTP client).
func isPrivateHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

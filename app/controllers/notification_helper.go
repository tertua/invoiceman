package controllers

import (
	"encoding/json"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// notifPayload is the stable v1 envelope forwarded to user webhook
// endpoints (n8n, GOWA bridges). event_id doubles as the downstream
// idempotency key; type selects the template on the consumer side.
type notifPayload struct {
	EventID    string    `json:"event_id"`
	Type       string    `json:"type"`
	Version    string    `json:"version"`
	OccurredAt string    `json:"occurred_at"`
	UserID     string    `json:"user_id"`
	Data       fiber.Map `json:"data"`
}

// enqueueNotification fans one event out to every subscribed, active
// endpoint of a user. Controllers call it after their write succeeds and
// return fast; the background worker forwards and retries.
func enqueueNotification(db *database.Queries, userID uuid.UUID, eventType, eventID string, data fiber.Map) {
	endpoints, err := db.ListActiveEndpoints(userID, eventType)
	if err != nil || len(endpoints) == 0 {
		return
	}
	if eventID == "" {
		eventID = "evt_" + uuid.NewString()[:8]
	}
	for _, e := range endpoints {
		enqueueNotificationDelivery(db, userID, e, eventType, eventID, data)
	}
}

// enqueueNotificationDelivery queues one event for one endpoint. The caller
// resolves eventID first so every endpoint in a fan-out shares the same
// payload event_id; the per-endpoint delivery key adds the endpoint suffix
// to satisfy the unique index.
func enqueueNotificationDelivery(db *database.Queries, userID uuid.UUID, endpoint models.NotificationEndpoint, eventType, eventID string, data fiber.Map) {
	raw, _ := json.Marshal(notifPayload{
		EventID:    eventID,
		Type:       eventType,
		Version:    "v1",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		UserID:     userID.String(),
		Data:       data,
	})
	_ = db.EnqueueDelivery(&models.NotificationDelivery{
		UserID:     userID,
		EndpointID: endpoint.ID,
		EventID:    eventID + ":" + endpoint.ID.String(),
		EventType:  eventType,
		TargetURL:  endpoint.TargetURL,
		Payload:    string(raw),
	})
}

// invoiceNotifData builds the data block shared by invoice.* events.
func invoiceNotifData(db *database.Queries, userID, invoiceID uuid.UUID) fiber.Map {
	out := fiber.Map{"invoice_id": invoiceID.String()}
	invoice, err := db.GetInvoice(userID, invoiceID)
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
		if client, err := db.GetClient(userID, *invoice.ClientID); err == nil {
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
	for k, v := range invoiceNotifData(db, payment.UserID, payment.InvoiceID) {
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

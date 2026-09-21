package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/relay"
)

// RelayPayload is the normalized v1 payload forwarded to downstream projects.
// Gateway identifies the provider (midtrans today, crypto tomorrow);
// fiat flows use gross_amount_idr, fractional flows use amount_decimal+currency.
type RelayPayload struct {
	EventID         string  `json:"event_id"`
	OrderID         string  `json:"order_id"`
	ProjectSlug     string  `json:"project_slug"`
	Gateway         string  `json:"gateway"`
	ExternalOrderID string  `json:"external_order_id"`
	Status          string  `json:"status"`
	GrossAmountIDR  float64 `json:"gross_amount_idr"`
	AmountDecimal   string  `json:"amount_decimal"`
	Currency        string  `json:"currency"`
	TransactionID   string  `json:"transaction_id"`
	PaymentType     string  `json:"payment_type"`
	PaidAt          string  `json:"paid_at"`
}

func terminalStatus(s string) bool {
	switch s {
	case models.GatewayStatusSuccess, models.GatewayStatusFailed,
		models.GatewayStatusExpired, models.GatewayStatusRefunded:
		return true
	default:
		return false
	}
}

// HandleMidtransWebhook is the single Midtrans notification URL.
// @Description Handle Midtrans payment notification.
// @Summary midtrans webhook
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param request body object true "Midtrans notification"
// @Success 200 {object} map[string]interface{}
// @Router /webhooks/midtrans [post]
func HandleMidtransWebhook(c fiber.Ctx) error {
	return handleGatewayWebhook(c, "midtrans")
}

// HandleGatewayWebhook handles notifications for any registered gateway.
// @Description Handle a payment gateway notification.
// @Summary gateway webhook
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param gateway path string true "Gateway name"
// @Param request body object true "Provider notification"
// @Success 200 {object} map[string]interface{}
// @Router /webhooks/{gateway} [post]
func HandleGatewayWebhook(c fiber.Ctx) error {
	return handleGatewayWebhook(c, strings.ToLower(strings.TrimSpace(c.Params("gateway"))))
}

func handleGatewayWebhook(c fiber.Ctx, gatewayName string) error {
	gw, err := gateway.Get(gatewayName)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "unknown payment gateway", nil)
	}
	raw := append([]byte(nil), c.Body()...)
	notif, err := gw.ParseAndVerify(raw)
	if err != nil {
		switch {
		case errors.Is(err, gateway.ErrNotConfigured):
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		case errors.Is(err, gateway.ErrInvalidSignature):
			return utils.Fail(c, fiber.StatusUnauthorized, "invalid signature", nil)
		default:
			return utils.Fail(c, fiber.StatusBadRequest, "invalid notification", nil)
		}
	}
	status := notif.Status

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	eventID, _ := randHex(8)
	_ = db.CreateEvent(&models.GatewayEvent{
		ID:        uuid.New(),
		OrderID:   notif.OrderID,
		Source:    gatewayName,
		Verified:  true,
		Payload:   string(raw),
		CreatedAt: time.Now(),
	})

	txn, err := db.GetTransaction(notif.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "unknown order", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load transaction", nil)
	}

	statusChanged := txn.Status != status
	// Idempotent: same status + same gateway txn id needs no further work,
	// unless no delivery ever succeeded (forward may have failed earlier).
	if !statusChanged && txn.MidtransTxnID == notif.TransactionID {
		deliveries, _ := db.ListDeliveriesByOrder(txn.OrderID)
		for _, d := range deliveries {
			if d.Status == "delivered" {
				return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true})
			}
		}
	}

	txn.Status = status
	txn.MidtransTxnID = notif.TransactionID
	txn.PaymentType = notif.PaymentType
	txn.RawNotification = string(raw)
	txn.UpdatedAt = time.Now()
	if status == models.GatewayStatusSuccess {
		now := time.Now()
		txn.PaidAt = &now
	}
	if err := db.SaveTransaction(&txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update transaction", nil)
	}

	// Settle local invoices on success.
	if status == models.GatewayStatusSuccess && txn.InvoiceID != nil && txn.UserID != nil {
		settleLocalInvoice(*db, txn, float64(notif.GrossMinor))
	}

	// Local orders have no downstream project to notify.
	if txn.ProjectSlug == "local" {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true})
	}

	project, err := db.GetProjectBySlug(txn.ProjectSlug)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "project not found", nil)
	}
	if !project.IsActive || strings.TrimSpace(project.WebhookURL) == "" {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true})
	}

	payload, _ := json.Marshal(RelayPayload{
		EventID:         "evt_" + eventID,
		OrderID:         txn.OrderID,
		ProjectSlug:     txn.ProjectSlug,
		Gateway:         gatewayName,
		ExternalOrderID: txn.ExternalOrderID,
		Status:          status,
		GrossAmountIDR:  float64(notif.GrossMinor),
		AmountDecimal:   notif.GrossDecimal,
		Currency:        notif.Currency,
		TransactionID:   notif.TransactionID,
		PaymentType:     notif.PaymentType,
		PaidAt:          time.Now().UTC().Format(time.RFC3339),
	})
	delivery := &models.WebhookDelivery{
		ID:          uuid.New(),
		OrderID:     txn.OrderID,
		ProjectSlug: txn.ProjectSlug,
		Gateway:     gatewayName,
		TargetURL:   project.WebhookURL,
		Payload:     string(payload),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Attempt:     1,
	}
	delivery.Signature = relay.SignPayload(payload, project.WebhookSecret)
	result, ferr := relay.Forward(c.Context(), project.WebhookURL, project.Slug, "evt_"+eventID, payload, project.WebhookSecret)
	if ferr != nil {
		retryAt := time.Now().Add(5 * time.Minute)
		delivery.Status = "failed"
		delivery.RespBody = truncateErr(ferr.Error())
		delivery.NextRetryAt = &retryAt
		_ = db.CreateDelivery(delivery)
		// Return success to Midtrans (we persist + retry ourselves);
		// the downstream retry is tracked in webhook_deliveries.
		return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true, "relayed": false})
	}
	delivery.RespCode = result.StatusCode
	delivery.RespBody = result.Body
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		delivery.Status = "delivered"
	} else {
		retryAt := time.Now().Add(5 * time.Minute)
		delivery.Status = "failed"
		delivery.NextRetryAt = &retryAt
	}
	_ = db.CreateDelivery(delivery)
	return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true, "relayed": delivery.Status == "delivered"})
}

func truncateErr(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}

// gatewayDisplayName maps a registry name to a human payment method label.
func gatewayDisplayName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "nowpayments":
		return "NOWPayments"
	case "", "midtrans":
		return "Midtrans"
	default:
		return strings.TrimSpace(name)
	}
}

// settleLocalInvoice records a payment against a local invoice on success.
// It is best-effort and idempotent via the invoice balance check.
func settleLocalInvoice(db database.Queries, txn models.GatewayTransaction, gross float64) {
	if txn.InvoiceID == nil || txn.UserID == nil {
		return
	}
	invoice, err := db.GetInvoice(*txn.UserID, *txn.InvoiceID)
	if err != nil {
		return
	}
	paid, err := db.PaidAmount(*txn.InvoiceID)
	if err != nil {
		return
	}
	balance := invoice.Total - paid
	if balance <= 0 {
		return
	}
	amount := balance
	if gross > 0 && gross < balance {
		amount = gross
	}
	now := time.Now()
	_ = db.CreatePayment(&models.Payment{
		ID:        uuid.New(),
		CreatedAt: now,
		UserID:    *txn.UserID,
		InvoiceID: *txn.InvoiceID,
		Amount:    amount,
		Method:    gatewayDisplayName(txn.Gateway),
		PaidOn:    &now,
		TxnID:     txn.MidtransTxnID,
		Notes:     gatewayDisplayName(txn.Gateway) + " " + txn.OrderID,
	})
}

// ListDeliveries returns recent relay deliveries for admins.
// @Description List relay deliveries.
// @Summary list relay deliveries
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/deliveries [get]
func ListDeliveries(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	rows, err := db.ListRecentDeliveries(100)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load deliveries", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"deliveries": deliveryResponses(rows)})
}

// ListMyDeliveries returns deliveries for one order owned by the calling project.
// @Description List own relay deliveries.
// @Summary list own deliveries
// @Tags Gateway
// @Produce json
// @Param order_id query string true "Global order ID"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/deliveries [get]
func ListMyDeliveries(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	orderID := strings.TrimSpace(c.Query("order_id"))
	if orderID == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "order_id is required", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	txn, err := db.GetTransaction(orderID)
	if err != nil || txn.ProjectSlug != project.Slug {
		return utils.Fail(c, fiber.StatusNotFound, "transaction not found", nil)
	}
	rows, err := db.ListDeliveriesByOrder(orderID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load deliveries", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"deliveries": deliveryResponses(rows)})
}

// RetryDelivery re-attempts a failed relay delivery.
// Service keys may retry only their own project; admins may retry any.
// @Description Retry a relay delivery.
// @Summary retry relay delivery
// @Tags Admin
// @Produce json
// @Param id path string true "Delivery ID"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/deliveries/{id}/retry [post]
func RetryDelivery(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid delivery id", nil)
	}
	delivery, err := db.GetDelivery(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "delivery not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load delivery", nil)
	}
	project, err := db.GetProjectBySlug(delivery.ProjectSlug)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "project not found", nil)
	}
	payload := []byte(delivery.Payload)
	signature := relay.SignPayload(payload, project.WebhookSecret)
	result, ferr := relay.Forward(c.Context(), delivery.TargetURL, project.Slug, "evt_retry_"+delivery.ID.String(), payload, project.WebhookSecret)
	delivery.Attempt++
	delivery.Signature = signature
	delivery.UpdatedAt = time.Now()
	if ferr != nil {
		retryAt := time.Now().Add(5 * time.Minute)
		delivery.Status = "failed"
		delivery.RespBody = truncateErr(ferr.Error())
		delivery.NextRetryAt = &retryAt
		_ = db.SaveDelivery(&delivery)
		return utils.Fail(c, fiber.StatusBadGateway, "retry failed", nil)
	}
	delivery.RespCode = result.StatusCode
	delivery.RespBody = result.Body
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		delivery.Status = "delivered"
		delivery.NextRetryAt = nil
	} else {
		retryAt := time.Now().Add(5 * time.Minute)
		delivery.Status = "failed"
		delivery.NextRetryAt = &retryAt
	}
	_ = db.SaveDelivery(&delivery)
	return utils.OK(c, fiber.StatusOK, fiber.Map{"delivery": deliveryResponse(delivery)})
}

func deliveryResponse(d models.WebhookDelivery) fiber.Map {
	return fiber.Map{
		"id":           d.ID,
		"order_id":     d.OrderID,
		"project_slug": d.ProjectSlug,
		"gateway":      d.Gateway,
		"target_url":   d.TargetURL,
		"attempt":      d.Attempt,
		"status":       d.Status,
		"resp_code":    d.RespCode,
		"created_at":   d.CreatedAt,
		"updated_at":   d.UpdatedAt,
	}
}

func deliveryResponses(rows []models.WebhookDelivery) []fiber.Map {
	out := make([]fiber.Map, 0, len(rows))
	for _, d := range rows {
		out = append(out, deliveryResponse(d))
	}
	return out
}

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
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/cache"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/relay"
)

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
	event := &models.GatewayEvent{
		ID:        uuid.New(),
		OrderID:   notif.OrderID,
		Source:    gatewayName,
		Verified:  true,
		Payload:   string(raw),
		CreatedAt: time.Now(),
	}
	if err := db.CreateEvent(event); err != nil {
		logger.L().Error("webhook event audit failed", "order_id", notif.OrderID, "gateway", gatewayName, "err", err)
	}

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
		if deliveries, derr := db.ListDeliveriesByOrder(txn.OrderID); derr == nil {
			for _, d := range deliveries {
				if d.Status == "delivered" {
					return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true})
				}
			}
		}
	}

	txn.Status = status
	txn.MidtransTxnID = notif.TransactionID
	txn.PaymentType = notif.PaymentType
	if txn.PaymentMethod == "" {
		txn.PaymentMethod = notif.PaymentMethod
	}
	txn.RawNotification = string(raw)
	txn.UpdatedAt = time.Now()
	if status == models.GatewayStatusSuccess {
		now := time.Now()
		txn.PaidAt = &now
	}
	// Validate webhook amount and currency before settlement to prevent money loss.
	if status == models.GatewayStatusSuccess && txn.ProjectSlug == "local" && txn.InvoiceID != nil && txn.UserID != nil {
		if verr := checkWebhookAmount(c, txn, notif); verr != nil {
			return verr
		}
		if err := db.SaveTransactionAndSettleInvoice(&txn, gateway.SettleAmount(models.MoneyFromMinor(notif.GrossMinor), notif.Currency, txn.InvoiceCurrency, txn.UsdToIdr), gatewayDisplayName(gatewayName)); err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to settle invoice payment", nil)
		}
		if err := cache.InvalidateUser(c.Context(), txn.UserID.String()); err != nil {
			logger.L().Warn("cache invalidation failed after payment settlement", "user_id", txn.UserID.String(), "order_id", notif.OrderID, "err", err)
		}
		eventHex, _ := randHex(8)
		enqueueNotification(db, *txn.UserID, models.NotifEventInvoiceStatusUpdated, "evt_"+eventHex,
			invoiceNotifData(db, *txn.UserID, *txn.InvoiceID))
	} else if err := db.SaveTransaction(&txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update transaction", nil)
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

	payload, _ := json.Marshal(relay.Payload{
		EventID:         "evt_" + eventID,
		OrderID:         txn.OrderID,
		ProjectSlug:     txn.ProjectSlug,
		Gateway:         gatewayName,
		ExternalOrderID: txn.ExternalOrderID,
		Status:          status,
		GrossAmountIDR:  notif.GrossMinor,
		AmountDecimal:   notif.GrossDecimal,
		Currency:        notif.Currency,
		TransactionID:   notif.TransactionID,
		PaymentType:     notif.PaymentType,
		PaymentMethod:   txn.PaymentMethod,
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
		Status:      "pending",
	}
	delivery.Signature = relay.SignPayload(payload, project.WebhookSecret)
	if err := db.CreateDelivery(delivery); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to queue delivery", nil)
	}
	// The background worker forwards and retries; the provider gets a fast
	// acknowledgement and downstream state is tracked in webhook_deliveries.
	return utils.OK(c, fiber.StatusOK, fiber.Map{"success": true, "queued": true})
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

// ListDeliveries returns one page of recent relay deliveries for admins.
// @Description List relay deliveries.
// @Summary list relay deliveries
// @Tags Admin
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/deliveries [get]
func ListDeliveries(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListRecentDeliveries(paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load deliveries", nil)
	}
	total, err := db.CountDeliveries()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count deliveries", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"deliveries": deliveryResponses(rows), "meta": paging.Meta(total)})
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
		if serr := db.SaveDelivery(&delivery); serr != nil {
			logger.L().Warn("delivery retry state store failed", "delivery_id", delivery.ID.String(), "err", serr)
		}
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
	if serr := db.SaveDelivery(&delivery); serr != nil {
		logger.L().Warn("delivery retry state store failed", "delivery_id", delivery.ID.String(), "err", serr)
	}
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

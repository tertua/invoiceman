package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/logger"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/gateway"
)

// webhookAmountTolerance is the max minor-unit drift accepted between the
// stored intent and the provider notification (rounding at the edges).
const webhookAmountTolerance = 1

// checkWebhookAmount verifies the notification matches the stored intent
// before settlement. It returns a fiber error response when the webhook
// must not settle, nil when safe to proceed.
func checkWebhookAmount(c fiber.Ctx, txn models.GatewayTransaction, notif *gateway.NotificationResult) error {
	if txn.Currency != notif.Currency {
		logger.L().Error("webhook currency mismatch - refusing to settle",
			"order_id", notif.OrderID,
			"expected_currency", txn.Currency,
			"webhook_currency", notif.Currency,
			"expected_amount", txn.AmountIDR,
			"webhook_amount", notif.GrossMinor,
			"invoice_id", *txn.InvoiceID,
			"user_id", *txn.UserID)
		return utils.Fail(c, fiber.StatusBadRequest, "webhook currency mismatch", nil)
	}
	diff := txn.AmountIDR - notif.GrossMinor
	if diff < 0 {
		diff = -diff
	}
	if diff > webhookAmountTolerance {
		logger.L().Error("webhook amount mismatch - refusing to settle",
			"order_id", notif.OrderID,
			"currency", txn.Currency,
			"expected_amount", txn.AmountIDR,
			"webhook_amount", notif.GrossMinor,
			"diff", diff,
			"invoice_id", *txn.InvoiceID,
			"user_id", *txn.UserID)
		return utils.Fail(c, fiber.StatusBadRequest, "webhook amount mismatch", nil)
	}
	return nil
}

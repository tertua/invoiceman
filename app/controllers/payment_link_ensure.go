package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

// paymentMethodOnline is the informational "how to pay" choice that opts an
// invoice into an auto-created public payment link on Save & send.
const paymentMethodOnline = "Online"

// ensurePaymentLink outcomes, mapped onto the API contract by linkFail.
var (
	errLinkLoad     = errors.New("failed to load payment link")
	errLinkPaidLoad = errors.New("failed to load invoice payments")
	errLinkPaid     = errors.New("invoice is already paid")
	errLinkCreate   = errors.New("failed to create payment link")
)

// ensurePaymentLink returns the invoice's public payment link, minting one
// when the invoice is sent and still carries a balance. An existing link is
// returned unchanged, so callers stay idempotent.
func ensurePaymentLink(db database.Queries, userID uuid.UUID, invoice models.Invoice) (models.PaymentLink, error) {
	link, err := db.GetPaymentLinkForInvoice(invoice.ID, userID)
	if err == nil {
		return link, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.PaymentLink{}, fmt.Errorf("%w: %v", errLinkLoad, err)
	}
	if invoice.Status != models.InvoiceStatusSent {
		return models.PaymentLink{}, errLinkPaid
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return models.PaymentLink{}, fmt.Errorf("%w: %v", errLinkPaidLoad, err)
	}
	if !invoice.Total.GreaterThan(paid) {
		return models.PaymentLink{}, errLinkPaid
	}
	token, err := newPaymentToken()
	if err != nil {
		return models.PaymentLink{}, fmt.Errorf("%w: %v", errLinkCreate, err)
	}
	link = models.PaymentLink{Token: token, InvoiceID: invoice.ID, UserID: userID, CreatedAt: time.Now()}
	if err := db.CreatePaymentLink(&link); err != nil {
		return models.PaymentLink{}, fmt.Errorf("%w: %v", errLinkCreate, err)
	}
	return link, nil
}

// linkFail maps ensurePaymentLink outcomes onto the stable error contract of
// the share endpoints.
func linkFail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errLinkPaid):
		return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
	case errors.Is(err, errLinkLoad):
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment link", nil)
	case errors.Is(err, errLinkPaidLoad):
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	default:
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment link", nil)
	}
}

// autoOnlinePaymentLink mints the public payment link as part of creating an
// Online invoice, so Save & send lands on a detail page that already shows
// the link. Best-effort: a link failure never fails invoice creation — the
// manual share button remains as the fallback.
func autoOnlinePaymentLink(db database.Queries, userID uuid.UUID, invoice models.Invoice) {
	if invoice.PaymentMethod != paymentMethodOnline || invoice.Status != models.InvoiceStatusSent {
		return
	}
	_, _ = ensurePaymentLink(db, userID, invoice)
}

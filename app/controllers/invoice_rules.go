package controllers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// Invoice business-rule violations. These describe an invoice whose fields are
// well-formed but whose state is not allowed, so they map to 422 rather than
// 400. Add new rules here, not inline at each call site.
var (
	ErrClientRequiredToSend = errors.New("client is required to send an invoice")
	ErrPendingPayment       = errors.New("invoice has a pending payment")
	ErrInvoicePaid          = errors.New("invoice is already paid")
)

// invoiceRuleStatus maps a business-rule violation to its HTTP status; anything
// unrecognized is treated as a malformed request.
func invoiceRuleStatus(err error) int {
	if errors.Is(err, ErrClientRequiredToSend) || errors.Is(err, ErrPendingPayment) || errors.Is(err, ErrInvoicePaid) {
		return fiber.StatusUnprocessableEntity
	}
	return fiber.StatusBadRequest
}

// failInvoiceRule writes the envelope for a builder/rule error.
func failInvoiceRule(c fiber.Ctx, err error) error {
	return utils.Fail(c, invoiceRuleStatus(err), err.Error(), nil)
}

// validateInvoice enforces the rules that constrain the invoice as a whole,
// given its resolved status and client. It is the single place to add a new
// cross-field invariant.
func validateInvoice(status string, clientID *uuid.UUID) error {
	// Sending bills someone: a sent invoice must name the client it is
	// addressed to, otherwise it is a receivable with no owner.
	if status == models.InvoiceStatusSent && clientID == nil {
		return ErrClientRequiredToSend
	}
	return nil
}

// resolveClient parses the payload's client id and checks the send rule. It
// keeps buildInvoice free of both concerns.
func resolveClient(input *models.InvoiceInput) (*uuid.UUID, error) {
	var clientID *uuid.UUID
	if input.ClientID != nil && *input.ClientID != "" {
		parsed, err := uuid.Parse(*input.ClientID)
		if err != nil {
			return nil, errors.New("invalid client_id")
		}
		clientID = &parsed
	}
	if err := validateInvoice(input.Status, clientID); err != nil {
		return nil, err
	}
	return clientID, nil
}

// guardInvoiceStatus rejects a status change that breaks an invoice rule: a
// live payment in flight, or sending without a client. It returns a real error
// so callers can `return failInvoiceRule(c, err)`; never a nil-after-write.
func guardInvoiceStatus(db database.Queries, userID, id uuid.UUID, status string, clientID *uuid.UUID) error {
	if db.PendingInvoiceIDs(userID)[id] {
		return ErrPendingPayment
	}
	return validateInvoice(status, clientID)
}

// rejectLockedInvoice reports why an invoice refuses edits: a live payment
// in flight, or money already covering the total. A PaidAmount load failure
// is returned raw so the caller maps it to 500 instead of a rule status.
func rejectLockedInvoice(db database.Queries, userID, id uuid.UUID, existing models.Invoice) error {
	paid, err := db.PaidAmount(id)
	if err != nil {
		return err
	}
	if db.PendingInvoiceIDs(userID)[id] {
		return ErrPendingPayment
	}
	if isPaidLocked(existing.Status, existing.DueDate, existing.Total, paid) {
		return ErrInvoicePaid
	}
	return nil
}

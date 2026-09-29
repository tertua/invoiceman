package controllers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// Invoice business-rule violations; state violations map to 422, ownership to 403, transitions to 400.
var (
	ErrClientRequiredToSend = errors.New("client is required to send an invoice")
	ErrPendingPayment       = errors.New("invoice has a pending payment")
	ErrInvoicePaid          = errors.New("invoice is already paid")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrOwnerRequired        = errors.New("org.ownerRequired")
	ErrOrgNotMember         = errors.New("org.notMember")
)

// invoiceRuleStatus maps a business-rule violation to its HTTP status; anything unrecognized is a malformed request.
func invoiceRuleStatus(err error) int {
	if errors.Is(err, ErrClientRequiredToSend) || errors.Is(err, ErrPendingPayment) || errors.Is(err, ErrInvoicePaid) {
		return fiber.StatusUnprocessableEntity
	}
	if errors.Is(err, ErrOwnerRequired) || errors.Is(err, ErrOrgNotMember) {
		return fiber.StatusForbidden
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

// statusTransitions is the invoice state machine keyed "from>to"; true means only the org owner may make that move.
var statusTransitions = map[string]bool{
	"draft>pending": false, // staff submit a draft for approval
	"draft>sent":    false, // sending bills the client
	"sent>draft":    false, // pull a sent invoice back to editing
	"paid>sent":     false, // reopen path; guardPaidReopen still blocks a money-paid invoice
	"pending>sent":  true,  // owner approves
	"pending>draft": true,  // owner rejects
}

// checkStatusTransition applies the state machine; unmapped moves are invalid, same-status is an idempotent no-op, and leaving pending needs the owner role.
func checkStatusTransition(c fiber.Ctx, from, to string) error {
	if from == to {
		return nil
	}
	ownerOnly, ok := statusTransitions[from+">"+to]
	if !ok {
		return ErrInvalidTransition
	}
	if !ownerOnly {
		return nil
	}
	role, err := utils.CurrentOrgRole(c)
	if err != nil || role != models.RoleOwner {
		return ErrOwnerRequired
	}
	return nil
}

// guardInvoiceStatus rejects a status change that breaks an invoice rule: a live payment in flight, a move outside the state machine, or sending without a client.
func guardInvoiceStatus(c fiber.Ctx, db database.Queries, orgID, id uuid.UUID, existing models.Invoice, status string) error {
	if db.PendingInvoiceIDs(orgID)[id] {
		return ErrPendingPayment
	}
	if err := checkStatusTransition(c, existing.Status, status); err != nil {
		return err
	}
	return validateInvoice(status, existing.ClientID)
}

// rejectLockedInvoice reports why an invoice refuses edits: a live payment
// in flight, or money already covering the total. A PaidAmount load failure
// is returned raw so the caller maps it to 500 instead of a rule status.
func rejectLockedInvoice(db database.Queries, orgID, id uuid.UUID, existing models.Invoice) error {
	paid, err := db.PaidAmount(id)
	if err != nil {
		return err
	}
	if db.PendingInvoiceIDs(orgID)[id] {
		return ErrPendingPayment
	}
	if isPaidLocked(existing.Status, existing.DueDate, existing.Total, paid) {
		return ErrInvoicePaid
	}
	return nil
}

// guardInvoiceDelete lets staff remove only drafts; deleting any other status needs the owner role (route-level guards cannot see the invoice).
func guardInvoiceDelete(c fiber.Ctx, status string) error {
	role, err := utils.CurrentOrgRole(c)
	if err != nil {
		return ErrOrgNotMember
	}
	if status != models.InvoiceStatusDraft && role != models.RoleOwner {
		return ErrOwnerRequired
	}
	return nil
}

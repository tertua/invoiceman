package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// TestMain runs the controller tests against in-memory SQLite with an in-memory store, mirroring the route test setup.
func TestMain(m *testing.M) {
	os.Setenv("STAGE_STATUS", "dev")
	os.Setenv("JWT_SECRET_KEY", "test-secret")
	os.Setenv("JWT_SECRET_KEY_EXPIRE_MINUTES_COUNT", "15")
	os.Setenv("JWT_REFRESH_KEY", "test-refresh")
	os.Setenv("JWT_REFRESH_KEY_EXPIRE_HOURS_COUNT", "720")
	os.Setenv("SQL_DSN", "")
	os.Setenv("SQLITE_PATH", "file::memory:?cache=shared")
	os.Setenv("REDIS_HOST", "")
	if err := database.Migrate(); err != nil {
		panic("test migrate failed: " + err.Error())
	}
	os.Exit(m.Run())
}

// approvalTestApp wires the approval handlers behind a stub that mimics the OrgContext middleware.
func approvalTestApp(userID, orgID uuid.UUID, role string) *fiber.App {
	withOrg := func(c fiber.Ctx) error {
		c.Locals(utils.SessionUserIDKey, userID)
		c.Locals(utils.SessionOrgIDKey, orgID)
		c.Locals(utils.SessionOrgRoleKey, role)
		return c.Next()
	}
	app := fiber.New()
	app.Post("/invoices/:id/submit", withOrg, SubmitInvoice)
	app.Post("/invoices/:id/approve", withOrg, ApproveInvoice)
	app.Post("/invoices/:id/reject", withOrg, RejectInvoice)
	return app
}

// postApproval performs a POST and returns the status plus decoded JSON body.
func postApproval(t *testing.T, app *fiber.App, route string) (status int, body map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(""))
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	defer resp.Body.Close()
	body = map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

// approvalErrMessage pulls error.message out of a utils.Fail envelope.
func approvalErrMessage(body map[string]any) string {
	errObj, _ := body["error"].(map[string]any)
	message, _ := errObj["message"].(string)
	return message
}

// approvalInvoiceStatus reads the invoice status out of an approval response envelope.
func approvalInvoiceStatus(body map[string]any) string {
	invoice, _ := body["invoice"].(map[string]any)
	status, _ := invoice["status"].(string)
	return status
}

// approvalSeedUser inserts a plain user row and returns its id.
func approvalSeedUser(t *testing.T, email string) uuid.UUID {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user := models.User{ID: uuid.New(), Name: "Approval Tester", Email: email, PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	require.NoError(t, db.CreateUser(&user))
	return user.ID
}

// approvalSeedOrg creates an org owned by the user and returns its id.
func approvalSeedOrg(t *testing.T, userID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	org, err := db.CreateOrg(name)
	require.NoError(t, err)
	require.NoError(t, db.CreateOwner(org.ID, userID))
	return org.ID
}

// approvalSeedInvoice inserts an invoice in the given status, optionally addressed to a client.
func approvalSeedInvoice(t *testing.T, orgID, userID uuid.UUID, status string, withClient bool) uuid.UUID {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	now := time.Now()
	invoice := &models.Invoice{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: &now,
		UserID:    userID,
		OrgID:     orgID,
		Status:    status,
		IssueDate: &now,
		DueDate:   &now,
		Currency:  "IDR",
		Subtotal:  decimal.RequireFromString("100000"),
		Total:     decimal.RequireFromString("100000"),
	}
	if withClient {
		client := models.Client{ID: uuid.New(), UserID: userID, OrgID: orgID, Name: "Approval Client"}
		require.NoError(t, db.CreateClient(&client))
		invoice.ClientID = &client.ID
	}
	require.NoError(t, db.CreateInvoice(orgID, invoice, nil))
	return invoice.ID
}

// TestSubmitInvoiceMovesDraftToPending proves staff and owner can submit a draft and get the pending detail back.
func TestSubmitInvoiceMovesDraftToPending(t *testing.T) {
	userID := approvalSeedUser(t, "approval-submit@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Submit Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusDraft, true)

	app := approvalTestApp(userID, orgID, models.RoleStaff)
	status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/submit")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, models.InvoiceStatusPending, approvalInvoiceStatus(body))

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	stored, err := db.GetInvoice(orgID, invoiceID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPending, stored.Status)
}

// TestSubmitInvoiceRejectsNonDraft proves only a draft may be submitted for approval.
func TestSubmitInvoiceRejectsNonDraft(t *testing.T) {
	userID := approvalSeedUser(t, "approval-submit-pending@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Submit Pending Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusPending, true)

	app := approvalTestApp(userID, orgID, models.RoleOwner)
	status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/submit")
	assert.Equal(t, fiber.StatusBadRequest, status, body)
	assert.Equal(t, "invalid status transition", approvalErrMessage(body))
}

// TestApproveInvoiceRequiresOwner proves a staff member cannot approve or reject.
func TestApproveInvoiceRequiresOwner(t *testing.T) {
	userID := approvalSeedUser(t, "approval-owner@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Owner Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusPending, true)

	app := approvalTestApp(userID, orgID, models.RoleStaff)
	for _, action := range []string{"approve", "reject"} {
		status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/"+action)
		assert.Equal(t, fiber.StatusForbidden, status, body)
		assert.Equal(t, "org.ownerRequired", approvalErrMessage(body))
	}
}

// TestApproveInvoiceRejectsMissingClient proves approving toward sent still enforces the send rule.
func TestApproveInvoiceRejectsMissingClient(t *testing.T) {
	userID := approvalSeedUser(t, "approval-client@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Client Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusPending, false)

	app := approvalTestApp(userID, orgID, models.RoleOwner)
	status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/approve")
	assert.Equal(t, fiber.StatusBadRequest, status, body)
	assert.Equal(t, "client is required to send an invoice", approvalErrMessage(body))
}

// TestApproveInvoiceSendsPendingInvoice proves the owner approve path flips pending to sent.
func TestApproveInvoiceSendsPendingInvoice(t *testing.T) {
	userID := approvalSeedUser(t, "approval-approve@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Approve Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusPending, true)

	app := approvalTestApp(userID, orgID, models.RoleOwner)
	status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/approve")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, models.InvoiceStatusSent, approvalInvoiceStatus(body))

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	stored, err := db.GetInvoice(orgID, invoiceID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, stored.Status)
}

// TestRejectInvoiceReturnsDraft proves the owner reject path sends a pending invoice back to draft.
func TestRejectInvoiceReturnsDraft(t *testing.T) {
	userID := approvalSeedUser(t, "approval-reject@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Reject Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusPending, true)

	app := approvalTestApp(userID, orgID, models.RoleOwner)
	status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/reject")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, models.InvoiceStatusDraft, approvalInvoiceStatus(body))

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	stored, err := db.GetInvoice(orgID, invoiceID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, stored.Status)
}

// TestApproveRejectsNonPending proves approve and reject only act on a pending invoice.
func TestApproveRejectsNonPending(t *testing.T) {
	userID := approvalSeedUser(t, "approval-transition@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Transition Org")
	invoiceID := approvalSeedInvoice(t, orgID, userID, models.InvoiceStatusDraft, true)

	app := approvalTestApp(userID, orgID, models.RoleOwner)
	for _, action := range []string{"approve", "reject"} {
		status, body := postApproval(t, app, "/invoices/"+invoiceID.String()+"/"+action)
		assert.Equal(t, fiber.StatusBadRequest, status, body)
		assert.Equal(t, "invalid status transition", approvalErrMessage(body))
	}
}

// TestApprovalGuardsIdAndOrg proves the shared preamble: 401 without org, 400 on a bad uuid, 404 on an unknown invoice.
func TestApprovalGuardsIdAndOrg(t *testing.T) {
	userID := approvalSeedUser(t, "approval-guards@example.com")
	orgID := approvalSeedOrg(t, userID, "Approval Guards Org")

	noOrg := approvalTestApp(userID, uuid.Nil, models.RoleOwner)
	status, body := postApproval(t, noOrg, "/invoices/"+uuid.NewString()+"/submit")
	assert.Equal(t, fiber.StatusUnauthorized, status, body)
	assert.Equal(t, "unauthorized, please sign in again", approvalErrMessage(body))

	app := approvalTestApp(userID, orgID, models.RoleOwner)
	status, body = postApproval(t, app, "/invoices/not-a-uuid/approve")
	assert.Equal(t, fiber.StatusBadRequest, status, body)
	assert.Equal(t, "invalid invoice id", approvalErrMessage(body))

	status, body = postApproval(t, app, "/invoices/"+uuid.NewString()+"/approve")
	assert.Equal(t, fiber.StatusNotFound, status, body)
	assert.Equal(t, "invoice not found", approvalErrMessage(body))
}

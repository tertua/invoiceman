package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestApp builds the full API app for flow tests.
func newTestApp() *fiber.App {
	app := fiber.New()
	PublicRoutes(app)
	GatewayRoutes(app)
	PrivateRoutes(app)
	return app
}

// doRequest performs a request against the test app.
func doRequest(t *testing.T, app *fiber.App, method, route, body string, cookies []*http.Cookie) *http.Response {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, route, reader)
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	return resp
}

// decodeBody decodes a JSON response body.
func decodeBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()

	data := map[string]interface{}{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&data))
	return data
}

// TestAuthFlow covers register, login, profile and password flows.
func TestAuthFlow(t *testing.T) {
	app := newTestApp()

	// Register a new user.
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Flow User","email":"flow@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 201, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "flow@example.com", body["user"].(map[string]interface{})["email"])
	assert.Equal(t, "admin", body["user"].(map[string]interface{})["role"])
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)

	// The next account starts as a regular user and can be promoted by admin.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Moderator User","email":"moderator@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	moderatorBody := decodeBody(t, resp)
	moderator := moderatorBody["user"].(map[string]interface{})
	assert.Equal(t, "user", moderator["role"])
	moderatorID := moderator["id"].(string)

	resp = doRequest(t, app, "GET", "/api/admin/users", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["users"].([]interface{}), 2)

	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+moderatorID+"/role", `{"role":"moderator"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "moderator", decodeBody(t, resp)["user"].(map[string]interface{})["role"])

	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"moderator@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	moderatorCookies := resp.Cookies()
	assert.Equal(t, "moderator", decodeBody(t, resp)["user"].(map[string]interface{})["role"])

	resp = doRequest(t, app, "GET", "/api/settings", "", moderatorCookies)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "PATCH", "/api/settings", `{"company_name":"Blocked"}`, moderatorCookies)
	assert.Equal(t, 403, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/admin/users", "", moderatorCookies)
	assert.Equal(t, 403, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/clients", `{"name":"Moderator Client"}`, moderatorCookies)
	assert.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	// Duplicate email is rejected.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Flow User","email":"flow@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 409, resp.StatusCode)
	resp.Body.Close()

	// Session user.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	assert.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Equal(t, "Flow User", body["user"].(map[string]interface{})["name"])

	// Update profile.
	resp = doRequest(t, app, "PATCH", "/api/auth/profile", `{"name":"Renamed"}`, cookies)
	assert.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Equal(t, "Renamed", body["user"].(map[string]interface{})["name"])

	// Wrong password is rejected.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"wrongpass"}`, nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// Login works.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	loginBody := decodeBody(t, resp)
	userID := loginBody["user"].(map[string]interface{})["id"].(string)
	cookies = resp.Cookies()

	// Expired access tokens are refreshed transparently via the refresh cookie.
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	expiredString, err := expiredToken.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))
	require.NoError(t, err)
	var refreshCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
		}
	}
	require.NotNil(t, refreshCookie)
	resp = doRequest(t, app, "GET", "/api/auth/me", "", []*http.Cookie{
		{Name: "access_token", Value: expiredString},
		refreshCookie,
	})
	assert.Equal(t, 200, resp.StatusCode)
	refreshedBody := decodeBody(t, resp)
	assert.Equal(t, "flow@example.com", refreshedBody["user"].(map[string]interface{})["email"])
	renewed := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "access_token" && cookie.Value != "" && cookie.Value != expiredString {
			renewed = true
		}
	}
	assert.True(t, renewed, "expected a fresh access_token cookie")
	cookies = resp.Cookies()

	// Change password.
	resp = doRequest(t, app, "PATCH", "/api/auth/password",
		`{"currentPassword":"secret123","newPassword":"newsecret123"}`, cookies)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)

	// Login with the new password.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"newsecret123"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)
	cookies = resp.Cookies()
	refreshCookie = nil
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
		}
	}
	require.NotNil(t, refreshCookie)

	// Forgot password always succeeds generically.
	resp = doRequest(t, app, "POST", "/api/auth/forgot-password",
		`{"email":"flow@example.com"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)

	// Reset with an unknown token fails.
	resp = doRequest(t, app, "POST", "/api/auth/reset-password",
		`{"token":"nope","new_password":"anothersecret123"}`, nil)
	assert.Equal(t, 400, resp.StatusCode)
	decodeBody(t, resp)

	// Logout ends the session: cookies are cleared and renewal is revoked.
	resp = doRequest(t, app, "POST", "/api/auth/logout", "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	cleared := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	assert.Contains(t, cleared, "access_token=;")
	assert.Contains(t, cleared, "refresh_token=;")
	resp.Body.Close()

	// The not-yet-expired access token is still cryptographically valid
	// (stateless JWT), but it can no longer be renewed after logout.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", []*http.Cookie{
		{Name: "access_token", Value: expiredString},
		refreshCookie,
	})
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()
}

// TestClientInvoiceFlow covers clients, invoices and dashboard.
func TestClientInvoiceFlow(t *testing.T) {
	app := newTestApp()

	// Register and keep the session cookies.
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Invoice User","email":"invoice@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Create a client.
	resp = doRequest(t, app, "POST", "/api/clients",
		`{"name":"Acme","email":"billing@acme.test","company":"Acme Inc"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	clientID := decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)
	require.NotEmpty(t, clientID)

	// Client detail starts empty.
	resp = doRequest(t, app, "GET", "/api/clients/"+clientID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)
	assert.Equal(t, float64(0), detail["stats"].(map[string]interface{})["count"])

	// Create an invoice: subtotal 250, discount 10, tax 10% -> total 264.
	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"client_id": "`+clientID+`",
		"status": "draft",
		"issue_date": "2026-09-01",
		"due_date": "2026-09-30",
		"currency": "USD",
		"tax_rate": 10,
		"discount": 10,
		"items": [
			{"description": "Design", "quantity": 2, "rate": 100},
			{"description": "Hosting", "quantity": 1, "rate": 50}
		]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoice := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, float64(264), invoice["total"])
	assert.Equal(t, "draft", invoice["effective_status"])
	assert.True(t, strings.HasPrefix(invoice["invoice_number"].(string), "INV-"))
	invoiceID := invoice["id"].(string)

	// List shows the invoice.
	resp = doRequest(t, app, "GET", "/api/invoices", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	listed := decodeBody(t, resp)["invoices"].([]interface{})
	require.Len(t, listed, 1)

	// Mark as sent.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+`/status`,
		`{"status":"sent"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "sent", updated["status"])

	// Dashboard reflects the new data.
	resp = doRequest(t, app, "GET", "/api/dashboard", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	dashboard := decodeBody(t, resp)
	stats := dashboard["stats"].(map[string]interface{})
	assert.Equal(t, float64(1), stats["invoiceCount"])
	assert.Equal(t, float64(1), stats["clientCount"])
	assert.Equal(t, float64(264), stats["outstanding"])
	assert.Len(t, dashboard["revenueSeries"].([]interface{}), 6)
	// Revenue points expose a stable YYYY-MM key for frontend localization.
	for _, item := range dashboard["revenueSeries"].([]interface{}) {
		point := item.(map[string]interface{})
		assert.Regexp(t, `^\d{4}-\d{2}$`, point["key"].(string))
		assert.NotEmpty(t, point["label"])
	}
	assert.Len(t, dashboard["recentInvoices"].([]interface{}), 1)

	// Currency filtering keeps dashboard totals from mixing currencies.
	resp = doRequest(t, app, "GET", "/api/dashboard?currency=IDR", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	idrDashboard := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(0), idrDashboard["invoiceCount"])

	resp = doRequest(t, app, "GET", "/api/dashboard?currency=USD", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	usdDashboard := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(1), usdDashboard["invoiceCount"])

	// Cleanup.
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+invoiceID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "DELETE", "/api/clients/"+clientID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()
}

// TestItemFlow covers catalog item CRUD and per-user ownership.
func TestItemFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Item User","email":"item@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/items",
		`{"name":"Consulting","description":"Hourly consulting","rate":125,"unit":"hour"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	item := decodeBody(t, resp)["item"].(map[string]interface{})
	itemID := item["id"].(string)
	assert.Equal(t, "Consulting", item["name"])

	resp = doRequest(t, app, "GET", "/api/items", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	items := decodeBody(t, resp)["items"].([]interface{})
	require.Len(t, items, 1)

	resp = doRequest(t, app, "PATCH", "/api/items/"+itemID,
		`{"name":"Strategy","description":"Strategy session","rate":200,"unit":"session"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["item"].(map[string]interface{})
	assert.Equal(t, "Strategy", updated["name"])
	assert.Equal(t, float64(200), updated["rate"])

	resp = doRequest(t, app, "DELETE", "/api/items/"+itemID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/items", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, decodeBody(t, resp)["items"])
}

// TestSettingsFlow covers settings defaults and updates.
func TestSettingsFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Settings User","email":"settings@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "GET", "/api/settings", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	settings := decodeBody(t, resp)["settings"].(map[string]interface{})
	assert.Equal(t, "IDR", settings["currency"])
	assert.Equal(t, "INV-", settings["invoice_prefix"])

	resp = doRequest(t, app, "PATCH", "/api/settings", `{
		"company_name":"Acme Studio",
		"email":"billing@acme.test",
		"phone":"+62123456789",
		"address":"Main Street",
		"logo_url":"data:image/png;base64,abc",
		"currency":"IDR",
		"tax_rate":11,
		"invoice_prefix":"ACME-"
	}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["settings"].(map[string]interface{})
	assert.Equal(t, "Acme Studio", updated["company_name"])
	assert.Equal(t, "IDR", updated["currency"])
	assert.Equal(t, float64(11), updated["tax_rate"])
	assert.Equal(t, "ACME-", updated["invoice_prefix"])

	resp = doRequest(t, app, "PATCH", "/api/settings", `{"currency":"INVALID"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}

// TestExpenseFlow covers expense CRUD, filtering, and totals.
func TestExpenseFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Expense User","email":"expense@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	today := time.Now().Format("2006-01-02")

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"Cloud Provider",
		"category":"Software",
		"expense_date":"`+today+`",
		"amount":100,
		"notes":"Monthly hosting"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	expense := decodeBody(t, resp)["expense"].(map[string]interface{})
	expenseID := expense["id"].(string)
	assert.Equal(t, "IDR", expense["currency"])

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"Office Supply",
		"category":"Office",
		"expense_date":"`+today+`",
		"amount":50,
		"currency":"IDR"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/expenses?category=Software", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Len(t, body["expenses"].([]interface{}), 1)
	assert.Equal(t, float64(100), body["totals"].(map[string]interface{})["total"])
	assert.Equal(t, float64(100), body["totals"].(map[string]interface{})["thisMonth"])

	resp = doRequest(t, app, "PATCH", "/api/expenses/"+expenseID, `{
		"vendor":"Cloud Provider Updated",
		"category":"Hosting",
		"expense_date":"`+today+`",
		"amount":125,
		"notes":"Updated"
	}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["expense"].(map[string]interface{})
	assert.Equal(t, "Hosting", updated["category"])
	assert.Equal(t, float64(125), updated["amount"])

	resp = doRequest(t, app, "DELETE", "/api/expenses/"+expenseID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()
}

// TestPaymentFlow covers recording, listing, balance validation, and deleting payments.
func TestPaymentFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Payment User","email":"payment@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"status":"sent",
		"issue_date":"2026-09-01",
		"due_date":"2026-09-30",
		"currency":"USD",
		"items":[{"description":"Service","quantity":1,"rate":100}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoice := decodeBody(t, resp)["invoice"].(map[string]interface{})
	invoiceID := invoice["id"].(string)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`",
		"amount":40,
		"method":"Bank transfer",
		"paid_on":"2026-09-21",
		"notes":"Deposit"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	payment := decodeBody(t, resp)["payment"].(map[string]interface{})
	paymentID := payment["id"].(string)

	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Len(t, body["payments"].([]interface{}), 1)
	assert.Equal(t, float64(40), body["totals"].(map[string]interface{})["total"])

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`",
		"amount":61,
		"method":"Cash",
		"paid_on":"2026-09-21"
	}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, float64(40), detail["paid_amount"])
	assert.Equal(t, "sent", detail["effective_status"])

	resp = doRequest(t, app, "DELETE", "/api/payments/"+paymentID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()
}

// TestReportsFlow covers report totals and breakdowns across features.
func TestReportsFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Reports User","email":"reports@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	today := time.Now().Format("2006-01-02")

	// Empty reports use arrays instead of null values for frontend safety.
	resp = doRequest(t, app, "GET", "/api/reports", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	emptyReport := decodeBody(t, resp)
	assert.NotNil(t, emptyReport["monthly"])
	assert.NotNil(t, emptyReport["aging"])
	assert.Empty(t, emptyReport["topClients"])
	assert.NotNil(t, emptyReport["statusBreakdown"])

	resp = doRequest(t, app, "POST", "/api/clients", `{"name":"Report Client"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	clientID := decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"client_id":"`+clientID+`",
		"status":"sent",
		"issue_date":"`+today+`",
		"due_date":"`+today+`",
		"currency":"USD",
		"items":[{"description":"Report work","quantity":1,"rate":100}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoiceID := decodeBody(t, resp)["invoice"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`","amount":40,"method":"Cash","paid_on":"`+today+`"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"Report Expense","category":"Software","expense_date":"`+today+`","amount":15
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/reports", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	report := decodeBody(t, resp)
	totals := report["totals"].(map[string]interface{})
	assert.Equal(t, float64(40), totals["revenue"])
	assert.Equal(t, float64(15), totals["expenses"])
	assert.Equal(t, float64(25), totals["netProfit"])
	assert.Equal(t, float64(60), totals["outstanding"])
	assert.Len(t, report["monthly"].([]interface{}), 6)
	for _, item := range report["monthly"].([]interface{}) {
		point := item.(map[string]interface{})
		assert.Regexp(t, `^\d{4}-\d{2}$`, point["key"].(string))
		assert.NotEmpty(t, point["label"])
	}
	assert.Len(t, report["aging"].([]interface{}), 5)
	// Aging buckets expose stable keys for frontend localization.
	agingKeys := make([]string, 0, 5)
	for _, item := range report["aging"].([]interface{}) {
		agingKeys = append(agingKeys, item.(map[string]interface{})["bucket"].(string))
	}
	assert.Equal(t, []string{"current", "d1_30", "d31_60", "d61_90", "d90_plus"}, agingKeys)
	assert.Len(t, report["topClients"].([]interface{}), 1)
	assert.Len(t, report["statusBreakdown"].([]interface{}), 4)
}

// TestPublicPaymentFlow covers public payment links and unconfigured gateway behavior.
func TestPublicPaymentFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Public User","email":"public@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"status":"sent",
		"issue_date":"2026-09-01",
		"due_date":"2026-09-30",
		"currency":"IDR",
		"items":[{"description":"Public service","quantity":1,"rate":100000}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoiceID := decodeBody(t, resp)["invoice"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	link := decodeBody(t, resp)
	token := link["token"].(string)
	assert.Contains(t, link["url"], "/pay/")

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	publicData := decodeBody(t, resp)
	assert.True(t, publicData["can_pay"].(bool))
	assert.Equal(t, invoiceID, publicData["invoice"].(map[string]interface{})["id"].(string))
	assert.Equal(t, "midtrans", publicData["gateway"].(map[string]interface{})["name"])

	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", "", nil)
	require.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token+"/status", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	status := decodeBody(t, resp)
	assert.NotEqual(t, "paid", status["status"])
	assert.NotEqual(t, float64(0), status["balance"])

	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", "", nil)
	assert.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/payments/online/send", `{"invoiceId":"`+invoiceID+`","email":"client@example.com"}`, cookies)
	assert.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()
}

// TestAIContractFlow covers auth, validation, and unconfigured-provider behavior.
func TestAIContractFlow(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/ai/business-summary", "", nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"AI User","email":"ai@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Valid requests return a clear configuration response without contacting Gemini.
	resp = doRequest(t, app, "POST", "/api/ai/business-summary", "", cookies)
	assert.Equal(t, 501, resp.StatusCode)
	assert.Contains(t, decodeBody(t, resp)["error"].(map[string]interface{})["message"], "not configured")

	resp = doRequest(t, app, "POST", "/api/ai/write-note", `{"kind":"invalid"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/ai/write-note", `{"kind":"terms"}`, cookies)
	assert.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/ai/payment-reminder", `{"invoiceId":"not-a-uuid","tone":"friendly"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/ai/receipt-parse", "", cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}

package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
	"github.com/tertua/invoiceman/platform/relay"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func sanitizeExternal(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 48 {
		out = out[:48]
	}
	if out == "" {
		out = "order"
	}
	return out
}

func randHex(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func relayOrderID(slug, external string) (string, error) {
	suffix, err := randHex(4)
	if err != nil {
		return "", err
	}
	return "EXT-" + slug + "-" + sanitizeExternal(external) + "-" + suffix, nil
}

func intentResponse(t models.GatewayTransaction) fiber.Map {
	return fiber.Map{
		"order_id":          t.OrderID,
		"external_order_id": t.ExternalOrderID,
		"project_slug":      t.ProjectSlug,
		"gateway":           t.Gateway,
		"amount_idr":        t.AmountIDR,
		"amount_decimal":    t.AmountDecimal,
		"currency":          t.Currency,
		"status":            t.Status,
		"snap_token":        t.SnapToken,
		"redirect_url":      t.RedirectURL,
		"payment_url":       t.PaymentURL,
		"address":           t.Address,
	}
}

// resolveGateway picks the provider: explicit request > project default > midtrans.
func resolveGateway(requested, projectDefault string) string {
	if name := strings.ToLower(strings.TrimSpace(requested)); name != "" {
		return name
	}
	if name := strings.ToLower(strings.TrimSpace(projectDefault)); name != "" {
		return name
	}
	return "midtrans"
}

// CreateIntent creates a Midtrans Snap transaction for a downstream project.
// Project identity is taken from the API key (GatewayAuth), never from the body.
// @Description Create a relayed payment intent.
// @Summary create payment intent
// @Tags Gateway
// @Accept json
// @Produce json
// @Param request body models.IntentInput true "Intent payload"
// @Success 201 {object} map[string]interface{}
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
// @Router /gateway/intents [post]
func CreateIntent(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	input := &models.IntentInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	if input.AmountIDR <= 0 && strings.TrimSpace(input.AmountDecimal) == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "amount_idr or amount_decimal is required", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	// Reuse a pending intent for the same external id + amount (safe retry).
	if existing, err := db.GetTransactionByExternal(project.Slug, input.ExternalOrderID); err == nil {
		if existing.Status == models.GatewayStatusPending && existing.AmountIDR == input.AmountIDR &&
			existing.AmountDecimal == strings.TrimSpace(input.AmountDecimal) && existing.SnapToken != "" {
			return utils.OK(c, fiber.StatusOK, intentResponse(existing))
		}
	}

	gatewayName := resolveGateway(input.Gateway, project.DefaultGateway)
	gw, err := gateway.Get(gatewayName)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "unknown payment gateway", nil)
	}
	orderID, err := relayOrderID(project.Slug, input.ExternalOrderID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create order", nil)
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = "IDR"
	}
	created, err := gw.CreateTransaction(c.Context(), &gateway.CreateTxRequest{
		OrderID:       orderID,
		AmountMinor:   input.AmountIDR,
		AmountDecimal: strings.TrimSpace(input.AmountDecimal),
		Currency:      currency,
		Email:         input.CustomerEmail,
		Phone:         input.CustomerPhone,
	})
	if err != nil {
		if errors.Is(err, gateway.ErrNotConfigured) {
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		}
		return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
	}
	raw, _ := json.Marshal(input)
	now := time.Now()
	txn := &models.GatewayTransaction{
		OrderID:         orderID,
		ProjectSlug:     project.Slug,
		Gateway:         gatewayName,
		ExternalOrderID: input.ExternalOrderID,
		AmountIDR:       input.AmountIDR,
		AmountDecimal:   strings.TrimSpace(input.AmountDecimal),
		Currency:        currency,
		CustomerEmail:   input.CustomerEmail,
		CustomerPhone:   input.CustomerPhone,
		Status:          models.GatewayStatusPending,
		SnapToken:       created.Token,
		RedirectURL:     created.RedirectURL,
		PaymentURL:      created.PaymentURL,
		Address:         created.Address,
		RawIntent:       string(raw),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateTransaction(txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	return utils.OK(c, fiber.StatusCreated, intentResponse(*txn))
}

// GetIntent returns one intent owned by the calling project.
// @Description Get a payment intent status.
// @Summary get payment intent
// @Tags Gateway
// @Produce json
// @Param order_id path string true "Global order ID"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/intents/{order_id} [get]
func GetIntent(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	txn, err := db.GetTransaction(c.Params("order_id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "transaction not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load transaction", nil)
	}
	if txn.ProjectSlug != project.Slug {
		return utils.Fail(c, fiber.StatusNotFound, "transaction not found", nil)
	}
	return utils.OK(c, fiber.StatusOK, intentResponse(txn))
}

// ListMyTransactions returns one page of recent intents for the calling project.
// @Description List own payment intents.
// @Summary list own intents
// @Tags Gateway
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/transactions [get]
func ListMyTransactions(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListTransactionsByProject(project.Slug, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load transactions", nil)
	}
	total, err := db.CountTransactionsByProject(project.Slug)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count transactions", nil)
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, t := range rows {
		out = append(out, intentResponse(t))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"transactions": out, "meta": paging.Meta(total)})
}

// GatewayConfig returns public browser configuration for the active gateway.
// Server credentials are never exposed here; the client key is intentionally
// public and is required by the provider's browser SDK.
// @Description Get public payment gateway browser configuration.
// @Summary get gateway browser config
// @Tags Gateway
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /gateway/config [get]
func GatewayConfig(c fiber.Ctx) error {
	cfg := midtrans.FromEnv()
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"gateway":       "midtrans",
		"client_key":    cfg.ClientKey,
		"is_production": cfg.IsProd,
		"configured":    cfg.ServerKey != "",
	})
}

// CreateInvoiceIntent creates a Snap transaction for a local invoice.
// Uses the session user (dashboard), not a service API key.
// @Description Create a Snap transaction for a local invoice.
// @Summary create invoice intent
// @Tags Gateway
// @Accept json
// @Produce json
// @Param request body map[string]string true "Invoice ID"
// @Success 201 {object} map[string]interface{}
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
// @Router /gateway/invoice-intents [post]
func CreateInvoiceIntent(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	var input struct {
		InvoiceID string `json:"invoiceId"`
	}
	if err := c.Bind().Body(&input); err != nil || input.InvoiceID == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "invoiceId is required", nil)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	invoice, err := db.GetInvoice(userID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	paid, err := db.PaidAmount(invoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	balance := invoice.Total - paid
	if balance <= 0 {
		return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
	}
	gw, err := gateway.Get("midtrans")
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "payment gateway is not registered", nil)
	}
	suffix, err := randHex(4)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create order", nil)
	}
	orderID := "INV-" + strings.ReplaceAll(invoice.InvoiceNumber, " ", "") + "-" + suffix
	amountIDR := int64(balance + 0.5)
	created, err := gw.CreateTransaction(c.Context(), &gateway.CreateTxRequest{
		OrderID:     orderID,
		AmountMinor: amountIDR,
		Currency:    invoice.Currency,
	})
	if err != nil {
		if errors.Is(err, gateway.ErrNotConfigured) {
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		}
		return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
	}
	now := time.Now()
	txn := &models.GatewayTransaction{
		OrderID:         orderID,
		ProjectSlug:     "local",
		Gateway:         "midtrans",
		ExternalOrderID: invoice.ID.String(),
		InvoiceID:       &invoice.ID,
		UserID:          &userID,
		AmountIDR:       amountIDR,
		Currency:        invoice.Currency,
		Status:          models.GatewayStatusPending,
		SnapToken:       created.Token,
		RedirectURL:     created.RedirectURL,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateTransaction(txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	return utils.OK(c, fiber.StatusCreated, intentResponse(*txn))
}

func projectResponse(p models.GatewayProject, revealSecrets bool, apiKey string) fiber.Map {
	out := fiber.Map{
		"slug":            p.Slug,
		"name":            p.Name,
		"webhook_url":     p.WebhookURL,
		"default_gateway": p.DefaultGateway,
		"is_active":       p.IsActive,
		"created_at":      p.CreatedAt,
		"updated_at":      p.UpdatedAt,
	}
	if revealSecrets {
		out["webhook_secret"] = p.WebhookSecret
	}
	if apiKey != "" {
		out["api_key"] = apiKey
	}
	return out
}

// CreateProject registers a downstream project and returns secrets once.
// @Description Register a downstream project.
// @Summary create gateway project
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body models.CreateProjectInput true "Project payload"
// @Success 201 {object} map[string]interface{}
// @Router /admin/gateway/projects [post]
func CreateProject(c fiber.Ctx) error {
	input := &models.CreateProjectInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	input.Slug = strings.TrimSpace(strings.ToLower(input.Slug))
	if !slugPattern.MatchString(input.Slug) {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid slug, use lowercase letters, numbers and dashes", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	defaultGateway := resolveGateway(input.DefaultGateway, "midtrans")
	if _, err := gateway.Get(defaultGateway); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "unknown payment gateway", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	if _, err := db.GetProjectBySlug(input.Slug); err == nil {
		return utils.Fail(c, fiber.StatusConflict, "project slug already exists", nil)
	}
	apiKey, err := relay.GenerateAPIKey()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to generate api key", nil)
	}
	secret, err := relay.GenerateSecret()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to generate secret", nil)
	}
	now := time.Now()
	p := &models.GatewayProject{
		Slug:           input.Slug,
		Name:           strings.TrimSpace(input.Name),
		APIKeyHash:     relay.HashKey(apiKey),
		WebhookURL:     strings.TrimSpace(input.WebhookURL),
		WebhookSecret:  secret,
		DefaultGateway: defaultGateway,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.CreateProject(p); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create project", nil)
	}
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"project": projectResponse(*p, true, apiKey)})
}

// ListProjects returns one page of downstream projects for admins.
// @Description List gateway projects.
// @Summary list gateway projects
// @Tags Admin
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/projects [get]
func ListProjects(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListProjects(paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load projects", nil)
	}
	total, err := db.CountProjects()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count projects", nil)
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, p := range rows {
		out = append(out, projectResponse(p, true, ""))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"projects": out, "meta": paging.Meta(total)})
}

// UpdateProject edits project metadata.
// @Description Update a gateway project.
// @Summary update gateway project
// @Tags Admin
// @Accept json
// @Produce json
// @Param slug path string true "Project slug"
// @Param request body models.UpdateProjectInput true "Project payload"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/projects/{slug} [patch]
func UpdateProject(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	p, err := db.GetProjectBySlug(c.Params("slug"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "project not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load project", nil)
	}
	input := &models.UpdateProjectInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	if strings.TrimSpace(input.Name) != "" {
		p.Name = strings.TrimSpace(input.Name)
	}
	if strings.TrimSpace(input.WebhookURL) != "" {
		p.WebhookURL = strings.TrimSpace(input.WebhookURL)
	}
	if strings.TrimSpace(input.DefaultGateway) != "" {
		name := resolveGateway(input.DefaultGateway, "")
		if _, err := gateway.Get(name); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "unknown payment gateway", nil)
		}
		p.DefaultGateway = name
	}
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}
	p.UpdatedAt = time.Now()
	if err := db.SaveProject(&p); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update project", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"project": projectResponse(p, true, "")})
}

// RotateProjectKey issues a new API key for a project (old key stops working).
// @Description Rotate a project API key.
// @Summary rotate project key
// @Tags Admin
// @Produce json
// @Param slug path string true "Project slug"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/projects/{slug}/rotate-key [post]
func RotateProjectKey(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	p, err := db.GetProjectBySlug(c.Params("slug"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "project not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load project", nil)
	}
	apiKey, err := relay.GenerateAPIKey()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to generate api key", nil)
	}
	p.APIKeyHash = relay.HashKey(apiKey)
	p.UpdatedAt = time.Now()
	if err := db.SaveProject(&p); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to rotate key", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"project": projectResponse(p, true, apiKey)})
}

// RotateProjectSecret issues a new webhook secret for a project.
// @Description Rotate a project webhook secret.
// @Summary rotate project secret
// @Tags Admin
// @Produce json
// @Param slug path string true "Project slug"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/projects/{slug}/rotate-secret [post]
func RotateProjectSecret(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	p, err := db.GetProjectBySlug(c.Params("slug"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "project not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load project", nil)
	}
	secret, err := relay.GenerateSecret()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to generate secret", nil)
	}
	p.WebhookSecret = secret
	p.UpdatedAt = time.Now()
	if err := db.SaveProject(&p); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to rotate secret", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"project": projectResponse(p, true, "")})
}

// ListAllTransactions returns one page of relay transactions for admins.
// @Description List relay transactions.
// @Summary list relay transactions
// @Tags Admin
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/transactions [get]
func ListAllTransactions(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListAllTransactions(paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load transactions", nil)
	}
	total, err := db.CountAllTransactions()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count transactions", nil)
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, t := range rows {
		out = append(out, intentResponse(t))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"transactions": out, "meta": paging.Meta(total)})
}

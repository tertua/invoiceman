package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/ai"
	"github.com/tertua/invoiceman/platform/database"
)

const maxReceiptSize = 10 << 20

func aiError(c fiber.Ctx, err error) error {
	if errors.Is(err, ai.ErrNotConfigured) {
		return utils.Fail(c, fiber.StatusNotImplemented, "AI provider is not configured", nil)
	}
	return utils.Fail(c, fiber.StatusBadGateway, "AI provider request failed", nil)
}

func parseAIJSON(text string, target interface{}) error {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return json.Unmarshal([]byte(strings.TrimSpace(text)), target)
}

// ReceiptParse extracts structured expense or invoice data from an uploaded receipt.
// @Description Parse a receipt with Gemini.
// @Summary parse receipt
// @Tags AI
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Receipt image or PDF"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /ai/receipt-parse [post]
func ReceiptParse(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt file is required", nil)
	}
	if file.Size > maxReceiptSize {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt file is too large", nil)
	}
	mimeType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(mimeType, "image/") && mimeType != "application/pdf" {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt must be an image or PDF", nil)
	}
	reader, err := file.Open()
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read receipt file", nil)
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(io.LimitReader(reader, maxReceiptSize+1))
	if err != nil || len(data) > maxReceiptSize {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read receipt file", nil)
	}
	result, err := ai.NewGeminiClient().GenerateWithFile(context.Background(), `Extract this receipt into JSON. Return only an object with these fields: vendor (string), category (string), date (YYYY-MM-DD string or empty), total (number), subtotal (number), notes (string), lineItems (array of objects with description (string), quantity (number), rate (number)). Use 0 for unknown numbers and empty strings for unknown text.`, mimeType, data)
	if err != nil {
		return aiError(c, err)
	}
	parsed := map[string]interface{}{}
	if err := parseAIJSON(result, &parsed); err != nil {
		return utils.Fail(c, fiber.StatusBadGateway, "AI provider returned invalid receipt data", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"result": parsed})
}

// BusinessSummary generates a financial summary for the current user.
// @Description Generate an AI business summary.
// @Summary generate business summary
// @Tags AI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /ai/business-summary [post]
func BusinessSummary(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	stats, err := db.GetStats(userID, "")
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load business data", nil)
	}
	report, err := db.GetReports(userID, "")
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load business data", nil)
	}
	input, _ := json.Marshal(fiber.Map{"stats": stats, "totals": report.Totals, "statusBreakdown": report.StatusBreakdown})
	result, err := ai.NewGeminiClient().Generate(context.Background(), "Write a concise, actionable business health summary in 2-4 sentences based on this JSON. Do not invent facts. JSON: "+string(input))
	if err != nil {
		return aiError(c, err)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"summary": result})
}

// PaymentReminder drafts a payment reminder for an invoice.
// @Description Generate an AI payment reminder.
// @Summary generate payment reminder
// @Tags AI
// @Accept json
// @Produce json
// @Param request body models.PaymentReminderInput true "Reminder payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /ai/payment-reminder [post]
func PaymentReminder(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.PaymentReminderInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	if _, err := db.GetInvoice(userID, invoiceID); err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	detail, err := invoiceDetail(*db, userID, invoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	payload, _ := json.Marshal(detail)
	result, err := ai.NewGeminiClient().GenerateJSON(context.Background(), "Create a payment reminder email as JSON with subject and body fields. Tone: "+input.Tone+". Be professional and concise. Invoice data: "+string(payload))
	if err != nil {
		return aiError(c, err)
	}
	draft := map[string]string{}
	if err := parseAIJSON(result, &draft); err != nil || draft["subject"] == "" || draft["body"] == "" {
		return utils.Fail(c, fiber.StatusBadGateway, "AI provider returned invalid reminder data", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"draft": draft})
}

// WriteNote writes invoice notes or terms using Gemini.
// @Description Generate invoice notes or terms.
// @Summary write invoice note
// @Tags AI
// @Accept json
// @Produce json
// @Param request body models.WriteNoteInput true "Note payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /ai/write-note [post]
func WriteNote(c fiber.Ctx) error {
	if _, err := utils.CurrentUserID(c); err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.WriteNoteInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	payload, _ := json.Marshal(input)
	result, err := ai.NewGeminiClient().Generate(context.Background(), "Write a polished invoice "+input.Kind+" in plain text. Keep it concise and suitable for a professional invoice. Use this JSON context: "+string(payload))
	if err != nil {
		return aiError(c, err)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"text": result})
}

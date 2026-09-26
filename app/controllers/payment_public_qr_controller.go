package controllers

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
)

// publicQrFetchTimeout bounds the server-side fetch of the provider QR image.
const publicQrFetchTimeout = 10 * time.Second

// publicQrMaxBytes caps the proxied QR image (provider QR png is kilobytes).
const publicQrMaxBytes = 1 << 20

// GetPublicQrImage proxies the current QRIS intent's QR image same-origin so
// the payer's download button saves the file instead of opening a new tab.
// The browser fetch→blob needs CORS headers the provider image host does not
// send; a same-origin GET needs none. Errors keep the utils.Fail envelope.
// @Description Get the current QRIS QR image for a public payment token.
// @Summary get public QRIS image
// @Tags Public Payments
// @Produce png
// @Param token path string true "Payment token"
// @Success 200 {string} binary "QR png"
// @Router /public/pay/{token}/qr [get]
func GetPublicQrImage(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	link, err := db.GetPaymentLink(c.Params("token"))
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "payment link not found", nil)
	}
	invoice, err := db.GetInvoice(link.UserID, link.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	if !invoice.Total.GreaterThan(paid) {
		return utils.Fail(c, fiber.StatusNotFound, "qr image not found", nil)
	}
	intent, err := db.LatestIntent("local", invoice.ID.String(), gateway.MethodQRIS)
	if err != nil || !reusableIntent(intent, invoice.Total.Sub(paid)) || strings.TrimSpace(intent.PaymentURL) == "" {
		return utils.Fail(c, fiber.StatusNotFound, "qr image not found", nil)
	}
	qrURL, err := url.Parse(strings.TrimSpace(intent.PaymentURL))
	if err != nil || (qrURL.Scheme != "http" && qrURL.Scheme != "https") || qrURL.Host == "" {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to load qr image", nil)
	}
	ctx, cancel := context.WithTimeout(c.Context(), publicQrFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qrURL.String(), nil)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to load qr image", nil)
	}
	req.Header.Set("Accept", "image/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to load qr image", nil)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to load qr image", nil)
	}
	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if !strings.HasPrefix(strings.ToLower(ct), "image/") {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to load qr image", nil)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, publicQrMaxBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > publicQrMaxBytes {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to load qr image", nil)
	}
	c.Set("Content-Type", ct)
	c.Set("Content-Disposition", "attachment; filename=\""+sanitizeExternal(intent.OrderID)+".png\"")
	c.Set("Cache-Control", "private, max-age=60")
	return c.Send(raw)
}

package controllers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/storage"
)

// GetSettings returns settings for the current user.
// @Description Get settings of current user.
// @Summary get current user settings
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /settings [get]
func GetSettings(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settings})
}

// UpdateSettings updates settings for the current user.
// @Description Update settings of current user.
// @Summary update current user settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.SettingsInput true "Settings payload"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /settings [patch]
func UpdateSettings(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.SettingsInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	settings.CompanyName = input.CompanyName
	settings.Email = input.Email
	settings.Phone = input.Phone
	settings.Address = input.Address
	settings.LogoURL = input.LogoURL
	settings.Currency = input.Currency
	settings.TaxRate = input.TaxRate
	settings.InvoicePrefix = input.InvoicePrefix
	if err := db.UpdateSettings(&settings); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update settings", nil)
	}
	recordAudit(c, db, userID, "settings.update", "settings", userID.String(), "")
	settings, err = db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load updated settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settings})
}

// maxLogoSize caps uploaded company logos (matches the SPA hint).
const maxLogoSize = 400 << 10

var allowedLogoTypes = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

// UploadLogo stores the company logo file and points LogoURL at it.
// Legacy data-URL and absolute values keep rendering unchanged; only new
// uploads go through storage.
// @Description Upload company logo.
// @Summary upload company logo
// @Tags Settings
// @Accept multipart/form-data
// @Produce json
// @Param logo formData file true "Logo image (PNG/JPEG/SVG, max 400KB)"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /settings/logo [post]
func UploadLogo(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	file, err := c.FormFile("logo")
	if err != nil || file == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "logo file is required", nil)
	}
	if file.Size > maxLogoSize {
		return utils.Fail(c, fiber.StatusBadRequest, "logo file is too large", nil)
	}
	reader, err := file.Open()
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read logo file", nil)
	}
	defer reader.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(reader, head)
	ct := http.DetectContentType(head[:n])
	ext, ok := allowedLogoTypes[strings.ToLower(strings.TrimSpace(ct))]
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "logo must be a PNG, JPEG, GIF, WEBP or SVG image", nil)
	}
	store, err := storage.Shared()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "file storage is not configured", nil)
	}
	key := storage.LogoKey(userID.String(), ext)
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read logo file", nil)
	}
	if err := store.Put(c.Context(), key, reader, file.Size, ct); err != nil {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to store logo file", nil)
	}
	settings.LogoURL = store.URLFor(key, storage.Public)
	if err := db.UpdateSettings(&settings); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update settings", nil)
	}
	recordAudit(c, db, userID, "settings.logo.upload", "settings", userID.String(), "")
	settings, err = db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load updated settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settings})
}

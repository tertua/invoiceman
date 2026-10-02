package controllers

import (
	"io"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/gateway"
	"github.com/tertua/tupay/platform/storage"
)

// GetSettings returns settings for the current user.
// @Description Get settings of current user.
// @Summary get current user settings
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /settings [get]
func GetSettings(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	settings, err := db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settingsPayload(settings)})
}

// UpdateSettings updates settings for the current user.
// @Description Update settings of current user. The provider_methods key is canonical; the deprecated midtrans_methods input alias is still accepted, and responses carry both keys.
// @Summary update current user settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.SettingsInput true "Settings payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /settings [patch]
func UpdateSettings(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
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
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	settings, err := db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	if input.UsdToIdr.IsNegative() {
		return utils.Fail(c, fiber.StatusBadRequest, "usd_to_idr cannot be negative", nil)
	}
	input.ProviderMethods = normalizeProviderMethods(gateway.DefaultProviderName, input.EffectiveProviderMethods())
	applySettingsInput(&settings, input)
	if err := db.UpdateSettings(&settings); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update settings", nil)
	}
	recordAudit(c, db, utils.CurrentActorID(c), "settings.update", "settings", orgID.String(), "")
	settings, err = db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load updated settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settingsPayload(settings)})
}

// UploadLogo stores the company logo file and points LogoURL at it.
// Legacy data-URL and absolute values keep rendering unchanged; only new
// uploads go through storage.
// @Description Upload company logo.
// @Summary upload company logo
// @Tags Settings
// @Accept multipart/form-data
// @Produce json
// @Param logo formData file true "Logo image (PNG/JPEG/GIF/WEBP, max 400KB)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /settings/logo [post]
func UploadLogo(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	settings, err := db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	const maxLogoSize = 400 << 10 // local constant, kept for clarity
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
	defer func() { _ = reader.Close() }()
	ct, ext, ok := utils.ValidateImage(reader, utils.LogoAllowedTypes)
	if !ok {
		// SVG is rejected by the helper (inline SVG executes scripts in the viewer's origin — stored XSS); other invalid types fail the allow list.
		return utils.Fail(c, fiber.StatusBadRequest, "logo must be a PNG, JPEG, GIF or WEBP image", nil)
	}
	store, err := storage.Shared()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "file storage is not configured", nil)
	}
	key := storage.LogoKey(orgID.String(), ext)
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
	recordAudit(c, db, utils.CurrentActorID(c), "settings.logo.upload", "settings", orgID.String(), "")
	settings, err = db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load updated settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settingsPayload(settings)})
}

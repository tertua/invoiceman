package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
)

// normalizeMidtransMethods keeps the settings PATCH write consistent with the
// routing below. Midtrans is locked to QRIS everywhere (code-first is the
// source of truth: the owner's account only offers QRIS), so any input
// normalizes to qris regardless of the stored per-owner value.
func normalizeMidtransMethods(string) string {
	return gateway.MethodQRIS
}

// midtransMethodAllowlist always allows exactly QRIS for Midtrans. The stored
// per-owner midtrans_methods value is intentionally ignored: no other Midtrans
// method is available on the owner's account, and the old DB value could not
// be trusted to reflect that.
func midtransMethodAllowlist() map[string]bool {
	return map[string]bool{gateway.MethodQRIS: true}
}

// enabledMidtransMethods always returns QRIS so the Snap fallback only ever
// offers the QRIS method (the Core API QRIS path is used directly first).
func enabledMidtransMethods() []string {
	return []string{gateway.MethodQRIS}
}

// applyEnabledMethods lets a Midtrans charge carry the method allowlist so
// Snap only offers QRIS. Other providers ignore it.
func applyEnabledMethods(spec *chargeSpec, provider string) {
	if provider != "midtrans" {
		return
	}
	spec.EnabledMethods = enabledMidtransMethods()
}

// ListMidtransMethods returns the Midtrans methods an owner can enable,
// provider-neutral, for the settings UI. It mirrors gateway.MethodName so the
// dashboard labels match the public pay page.
// @Description List selectable Midtrans payment methods.
// @Summary list midtrans methods
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /settings/methods [get]
func ListMidtransMethods(c fiber.Ctx) error {
	methods := make([]models.GatewayMethod, 0)
	for _, method := range (midtrans.Gateway{}).Methods() {
		methods = append(methods, models.GatewayMethod{ID: method, Name: gateway.MethodName(method)})
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"methods": methods})
}

package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/gateway"
	"github.com/tertua/tupay/platform/midtrans"
)

// methodAllowedFor reports whether a provider may offer a neutral method.
// Today only the default provider is narrowed by the owner allowlist; every
// other provider is unrestricted. Phase 3 replaces this with a per-provider
// allowlist resolver.
func methodAllowedFor(gw gateway.Gateway, method string, allow map[string]bool) bool {
	if gw.Name() != gateway.DefaultProvider() {
		return true
	}
	return methodAllowed(allow, method)
}

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

// applyEnabledMethods lets the built-in provider's charge carry the method
// allowlist so its checkout only offers QRIS. The constant (not the env-driven
// default) keeps the QRIS lock pinned to the account that is QRIS-only. Other
// providers ignore it.
func applyEnabledMethods(spec *chargeSpec, provider string) {
	if provider != gateway.DefaultProviderName {
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

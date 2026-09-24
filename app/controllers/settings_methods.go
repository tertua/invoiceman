package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
)

// normalizeMidtransMethods cleans a CSV of gateway ids down to the ones
// Midtrans actually supports, preserving request order and dropping
// duplicates. An empty result means "all methods", so unknown ids silently
// disappear rather than being stored.
func normalizeMidtransMethods(raw string) string {
	supported := make(map[string]bool)
	for _, method := range (midtrans.Gateway{}).Methods() {
		supported[method] = true
	}
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		method := strings.ToLower(strings.TrimSpace(part))
		if method == "" || seen[method] || !supported[method] {
			continue
		}
		seen[method] = true
		out = append(out, method)
	}
	return strings.Join(out, ",")
}

// midtransMethodAllowlist parses the stored CSV into a lookup the router
// consults. An empty list allows every method.
func midtransMethodAllowlist(settings models.Settings) map[string]bool {
	raw := strings.TrimSpace(settings.MidtransMethods)
	if raw == "" {
		return nil
	}
	allow := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		if method := strings.ToLower(strings.TrimSpace(part)); method != "" {
			allow[method] = true
		}
	}
	if len(allow) == 0 {
		return nil
	}
	return allow
}

// enabledMidtransMethods returns the allowlisted methods in Midtrans'
// canonical order, or nil when the owner never restricted them.
func enabledMidtransMethods(settings models.Settings) []string {
	allow := midtransMethodAllowlist(settings)
	if allow == nil {
		return nil
	}
	out := make([]string, 0, len(allow))
	for _, method := range (midtrans.Gateway{}).Methods() {
		if allow[method] {
			out = append(out, method)
		}
	}
	return out
}

// applyEnabledMethods lets a Midtrans charge carry the owner's method
// allowlist so Snap only offers the enabled ones. Other providers ignore it.
func applyEnabledMethods(spec *chargeSpec, provider string, settings models.Settings) {
	if provider != "midtrans" {
		return
	}
	spec.EnabledMethods = enabledMidtransMethods(settings)
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

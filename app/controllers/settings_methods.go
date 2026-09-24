package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
)

// normalizeMidtransMethods reduces a raw CSV to the single Midtrans method
// the settings dropdown stores. It keeps the first supported id and falls
// back to gopay for empty, unknown, or legacy multi-method values so the
// public pay page always offers exactly one method.
func normalizeMidtransMethods(raw string) string {
	for _, part := range strings.Split(raw, ",") {
		method := strings.ToLower(strings.TrimSpace(part))
		for _, supported := range (midtrans.Gateway{}).Methods() {
			if method == supported {
				return method
			}
		}
	}
	return gateway.MethodGopay
}

// midtransMethodAllowlist parses the stored single method id into a lookup
// the router consults. Empty (legacy) allows every method.
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

// enabledMidtransMethods returns the single configured method, defaulting to
// gopay for legacy empty settings so the public page always offers exactly
// one method.
func enabledMidtransMethods(settings models.Settings) []string {
	allow := midtransMethodAllowlist(settings)
	if allow == nil {
		return []string{gateway.MethodGopay}
	}
	out := make([]string, 0, len(allow))
	for _, method := range (midtrans.Gateway{}).Methods() {
		if allow[method] {
			out = append(out, method)
		}
	}
	if len(out) == 0 {
		return []string{gateway.MethodGopay}
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

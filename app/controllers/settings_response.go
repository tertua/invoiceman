package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
)

// settingsResponse renders settings for the API. The model's own json tags carry
// the provider-neutral keys; this helper additionally emits the DEPRECATED
// midtrans_methods alias so relay/dashboard clients that read the old key keep
// working until they migrate (mirrors how intentResponse keeps snap_token). It
// is a hand-built fiber.Map rather than a raw struct so both keys always agree.
func settingsResponse(s models.Settings) fiber.Map {
	return fiber.Map{
		"org_id":           s.OrgID,
		"user_id":          s.UserID,
		"updated_at":       s.UpdatedAt,
		"company_name":     s.CompanyName,
		"email":            s.Email,
		"phone":            s.Phone,
		"address":          s.Address,
		"logo_url":         s.LogoURL,
		"currency":         s.Currency,
		"tax_rate":         s.TaxRate,
		"usd_to_idr":       s.UsdToIdr,
		"provider_methods": s.ProviderMethods,
		"midtrans_methods": s.ProviderMethods,
		"invoice_prefix":   s.InvoicePrefix,
		"language":         s.Language,
	}
}

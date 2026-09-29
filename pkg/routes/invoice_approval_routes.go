package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerInvoiceApprovalRoutes wires the invoice approval workflow: submit by any member, approve/reject owner-only (D11).
func registerInvoiceApprovalRoutes(route fiber.Router) {
	route.Post("/invoices/:id/submit", controllers.SubmitInvoice)
	route.Post("/invoices/:id/approve", middleware.RequireOrgRole("owner"), controllers.ApproveInvoice)
	route.Post("/invoices/:id/reject", middleware.RequireOrgRole("owner"), controllers.RejectInvoice)
}

package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/controllers"
	"github.com/tertua/invoiceman/pkg/middleware"
)

// PrivateRoutes func for describe group of private routes.
func PrivateRoutes(a *fiber.App) {
	// Create routes group matching the frontend apiClient baseURL ("/api").
	route := a.Group("/api", middleware.AuthRequired())

	// Auth session routes:
	route.Get("/auth/me", controllers.Me)                     // get current session user
	route.Patch("/auth/profile", controllers.UpdateProfile)   // update display name
	route.Patch("/auth/password", controllers.ChangePassword) // change password
	route.Post("/auth/logout", controllers.Logout)            // end session

	// Client routes:
	route.Get("/clients", controllers.ListClients)         // get all clients
	route.Post("/clients", controllers.CreateClient)       // create a new client
	route.Get("/clients/:id", controllers.GetClient)       // get client with invoices and stats
	route.Patch("/clients/:id", controllers.UpdateClient)  // update a client
	route.Delete("/clients/:id", controllers.DeleteClient) // delete a client

	// Invoice routes:
	route.Get("/invoices", controllers.ListInvoices)                     // get invoices with filters
	route.Post("/invoices", controllers.CreateInvoice)                   // create a new invoice
	route.Get("/invoices/:id", controllers.GetInvoice)                   // get invoice with items and payments
	route.Patch("/invoices/:id", controllers.UpdateInvoice)              // update an invoice
	route.Patch("/invoices/:id/status", controllers.UpdateInvoiceStatus) // update invoice status
	route.Delete("/invoices/:id", controllers.DeleteInvoice)             // delete an invoice

	// Dashboard routes:
	route.Get("/dashboard", controllers.GetDashboard) // get dashboard aggregates
}

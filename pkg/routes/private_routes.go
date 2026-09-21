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

	// Catalog item routes:
	route.Get("/items", controllers.ListItems)
	route.Post("/items", controllers.CreateItem)
	route.Patch("/items/:id", controllers.UpdateItem)
	route.Delete("/items/:id", controllers.DeleteItem)

	// Expense routes:
	route.Get("/expenses", controllers.ListExpenses)
	route.Post("/expenses", controllers.CreateExpense)
	route.Patch("/expenses/:id", controllers.UpdateExpense)
	route.Delete("/expenses/:id", controllers.DeleteExpense)

	// Payment routes:
	route.Get("/payments", controllers.ListPayments)
	route.Post("/payments", controllers.CreatePayment)
	route.Delete("/payments/:id", controllers.DeletePayment)
	route.Post("/payments/online", controllers.CreateOnlineLink)
	route.Post("/payments/online/send", controllers.SendOnlineLink)

	// Reports routes:
	route.Get("/reports", controllers.GetReports)

	// Settings routes:
	route.Get("/settings", controllers.ListSettings)
	route.Patch("/settings", middleware.RequireRoles("admin", "user"), controllers.UpdateSettings)

	// AI routes:
	route.Post("/ai/receipt-parse", controllers.ReceiptParse)
	route.Post("/ai/business-summary", controllers.BusinessSummary)
	route.Post("/ai/payment-reminder", controllers.PaymentReminder)
	route.Post("/ai/write-note", controllers.WriteNote)

	// Local invoice Snap intents (session user, no service key needed).
	route.Post("/gateway/invoice-intents", controllers.CreateInvoiceIntent)

	admin := a.Group("/api/admin", middleware.AuthRequired(), middleware.RequireRoles("admin"))
	admin.Get("/users", controllers.ListUsers)
	admin.Patch("/users/:id/role", controllers.UpdateUserRole)
}

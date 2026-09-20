package routes

import (
	"github.com/gofiber/fiber/v3"

	swagger "github.com/gofiber/contrib/v3/swaggo"
)

// SwaggerRoute func for describe group of API Docs routes.
func SwaggerRoute(a *fiber.App) {
	// Create routes group.
	route := a.Group("/swagger")

	// Routes for GET method:
	route.Get("*", swagger.HandlerDefault) // get one user by ID
}

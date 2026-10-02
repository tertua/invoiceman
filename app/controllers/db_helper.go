package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// openDB opens the request-scoped database connection. On failure it writes a
// 500 response and returns ok=false; callers should do:
//
//	db, ok := openDB(c)
//	if !ok {
//	    return nil
//	}
func openDB(c fiber.Ctx) (*database.Queries, bool) {
	db, err := database.OpenDBConnection()
	if err != nil {
		_ = utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
		return nil, false
	}
	return db, true
}

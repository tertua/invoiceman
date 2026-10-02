package controllers

import (
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
)

// verifyEmailGate returns the account status a new registration should get:
// pending when verification is required and this is not the first-install
// bootstrap (count == 0), active otherwise.
func verifyEmailGate(count int64) int {
	if configs.Get().Auth.RequireEmailVerification && count > 0 {
		return models.UserStatusPending
	}
	return models.UserStatusActive
}

// accountStatusReason maps a non-active account status to its audit reason so
// the password and SSO gates report the same word for the same state.
func accountStatusReason(status int) string {
	if status == models.UserStatusPending {
		return "pending"
	}
	return "blocked"
}

// statusAllowed reports whether an account status may start a session.
func statusAllowed(status int) bool {
	return status == models.UserStatusActive
}

// gateAccountStatus rejects login for anything but an active account, writing
// the 403 itself and returning true when the request is done. It runs after
// password verify so a wrong password never leaks the account state. A false
// return means active: the caller may start the session.
func gateAccountStatus(c fiber.Ctx, db *database.Queries, user models.User) bool {
	if statusAllowed(user.UserStatus) {
		return false
	}
	reason := accountStatusReason(user.UserStatus)
	recordLoginFailure(c, db, user.Email, reason, user.ID)
	if reason == "pending" {
		_ = utils.Fail(c, fiber.StatusForbidden, "account is pending verification", nil)
	} else {
		_ = utils.Fail(c, fiber.StatusForbidden, "account is blocked", nil)
	}
	return true
}

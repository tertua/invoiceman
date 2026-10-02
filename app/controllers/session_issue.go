package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// startSession bundles the four steps every login/register/verify path needs:
// issue the session id/refresh tokens, mint the CSRF token, bind them in the
// session store, and persist the cookie. activeOrg can be empty when the
// caller will resolve it later.
//
// It returns the freshly issued token pair (*utils.Tokens, containing SID and
// Refresh) so a caller can pass the session id along if it needs to. Most
// callers ignore the return value: the cookie and the session store already
// carry the tokens.
//
// Callers should:
//
//	tokens, err := startSession(c, db, user.ID, loginOrgHint(db, user.ID))
//	if err != nil {
//		return utils.Fail(c, fiber.StatusInternalServerError, "failed to start session", nil)
//	}
//	_ = tokens // not needed externally; the cookie / session store carries them
func startSession(c fiber.Ctx, db *database.Queries, userID uuid.UUID, activeOrg string) (*utils.Tokens, error) {
	tokens, err := utils.IssueSession(c, userID, "")
	if err != nil {
		return nil, err
	}
	csrf, err := issueCSRF(c)
	if err != nil {
		return nil, err
	}
	if err := saveRefreshToken(c.Context(), userID, tokens.SID, tokens.Refresh, csrf, activeOrg); err != nil {
		return nil, err
	}
	return tokens, nil
}

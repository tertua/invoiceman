package routes

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// API version prefixes. New clients use V1; Legacy is kept working with
// Deprecation/Sunset headers (see DeprecationHeaders).
const (
	APILegacyPrefix = "/api"
	APIV1Prefix     = "/api/v1"
	// SunsetDate advertises when the legacy prefix stops working.
	SunsetDate = "Mon, 01 Mar 2027 00:00:00 GMT"
)

// RegisterAPI registers every versioned route group under prefix.
// GatewayRoutes stays before PrivateRoutes per prefix: the session
// AuthRequired group matches the prefix, so service routes registered
// after it would be forced through cookie sessions.
//
// ORDERING (Fiber gotcha): RegisterAPI(APIV1Prefix) MUST run before
// RegisterAPI(APILegacyPrefix). Group middleware mounts as a prefix Use,
// so the legacy private Use("/api", AuthRequired, ...) would otherwise
// shadow public routes registered later under /api/v1.
func RegisterAPI(a *fiber.App, prefix string) {
	PublicRoutesAt(a, prefix)
	GatewayRoutesAt(a, prefix)
	PrivateRoutesAt(a, prefix)
}

// DeprecationHeaders marks legacy /api (non-v1) responses so clients can
// migrate on a schedule. Applied globally; /api/v1 is never marked.
func DeprecationHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		p := c.Path()
		if strings.HasPrefix(p, APILegacyPrefix+"/") && !strings.HasPrefix(p, APIV1Prefix+"/") {
			c.Set("Deprecation", "true")
			c.Set("Sunset", SunsetDate)
		}
		return c.Next()
	}
}

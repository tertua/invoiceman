package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerOrgRoutes wires the organization endpoints on the private session group (OrgContext runs in the group chain).
func registerOrgRoutes(route fiber.Router) {
	route.Get("/orgs/me", controllers.GetMyOrg)
	route.Post("/orgs/:id/activate", controllers.ActivateOrg)
	route.Patch("/orgs/:id", middleware.RequireOrgRole("owner"), controllers.RenameOrg)
	route.Get("/orgs/members", controllers.ListMembers)
	route.Post("/orgs/invites", middleware.RequireOrgRole("owner"), controllers.CreateOrgInvite)
	route.Get("/orgs/invites", middleware.RequireOrgRole("owner"), controllers.ListOrgInvites)
	route.Delete("/orgs/invites/:id", middleware.RequireOrgRole("owner"), controllers.RevokeOrgInvite)
	route.Post("/orgs/invites/accept", controllers.AcceptOrgInvite)
}

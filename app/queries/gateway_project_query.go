package queries

import "github.com/tertua/tupay/app/models"

// DeleteProject removes one project row by slug; it stays slug-scoped because authorization lives in the admin route (RequireRoles("admin")), not here.
// Callers must guard on CountTransactionsByProject first: transactions are financial records and are never cascade-deleted.
func (q *GatewayQueries) DeleteProject(slug string) error {
	return q.Where("slug = ?", slug).Delete(&models.GatewayProject{}).Error
}

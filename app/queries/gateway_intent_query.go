package queries

import "github.com/tertua/invoiceman/app/models"

// LatestIntent returns the newest transaction for one project + external id +
// payment method, so a public pay page can find its current intent across
// retry suffixes (-r1, -r2, ...). order_id breaks created_at ties, keeping
// the newest suffix deterministic when two rows share a second.
func (q *GatewayQueries) LatestIntent(projectSlug, external, method string) (models.GatewayTransaction, error) {
	t := models.GatewayTransaction{}
	err := q.Where("project_slug = ? AND external_order_id = ? AND payment_method = ?",
		projectSlug, external, method).
		Order("created_at DESC, order_id DESC").First(&t).Error
	return t, notFound(err)
}

// CountIntents counts the transactions for one project + external id + method;
// the public pay page uses it to pick the next retry suffix (base + "-rN").
func (q *GatewayQueries) CountIntents(projectSlug, external, method string) (int64, error) {
	var n int64
	err := q.Model(&models.GatewayTransaction{}).
		Where("project_slug = ? AND external_order_id = ? AND payment_method = ?",
			projectSlug, external, method).
		Count(&n).Error
	return n, err
}

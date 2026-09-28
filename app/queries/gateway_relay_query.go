package queries

import "github.com/tertua/tupay/app/models"

// RelayClaimPrefix namespaces claim rows in gateway_transactions. A claim is
// an in-flight slot for a relayed intent charge, never a payable intent and
// never sent to a provider, so reuse decisions must look past these rows.
const RelayClaimPrefix = "CLM-"

// LatestRelayIntent returns the newest non-claim transaction for a project +
// external id, so safe retries reuse a live intent instead of charging again.
func (q *GatewayQueries) LatestRelayIntent(projectSlug, external string) (models.GatewayTransaction, error) {
	t := models.GatewayTransaction{}
	err := q.Where("project_slug = ? AND external_order_id = ? AND order_id NOT LIKE ?",
		projectSlug, external, RelayClaimPrefix+"%").
		Order("created_at DESC").First(&t).Error
	return t, notFound(err)
}

package queries

import (
	"database/sql"
	"strings"

	"github.com/tertua/tupay/app/models"
)

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

// intentBelongsToBase reports whether an order id belongs to one public pay
// base: the base itself or one of its retry suffixes (base-rN).
func intentBelongsToBase(orderID, base string) bool {
	return orderID == base || strings.HasPrefix(orderID, base+"-r")
}

// intentsForMethod loads an invoice's intents for one method, newest first,
// so callers can pick the newest row owned by a specific order-id base.
func (q *GatewayQueries) intentsForMethod(projectSlug, external, method string) ([]models.GatewayTransaction, error) {
	rows := []models.GatewayTransaction{}
	err := q.Where("project_slug = ? AND external_order_id = ? AND payment_method = ?",
		projectSlug, external, method).
		Order("created_at DESC, order_id DESC").Find(&rows).Error
	return rows, err
}

// LatestIntentForBase returns the newest transaction for one order-id base
// (the base or a base-rN retry). It keeps crypto assets apart, so switching
// assets opens a fresh deposit address instead of reusing another asset's.
func (q *GatewayQueries) LatestIntentForBase(projectSlug, external, method, base string) (models.GatewayTransaction, error) {
	rows, err := q.intentsForMethod(projectSlug, external, method)
	if err != nil {
		return models.GatewayTransaction{}, err
	}
	for _, r := range rows {
		if intentBelongsToBase(r.OrderID, base) {
			return r, nil
		}
	}
	return models.GatewayTransaction{}, sql.ErrNoRows
}

// CountIntentsForBase counts the transactions for one order-id base, so retry
// suffixes advance per asset instead of across every crypto asset of one
// invoice.
func (q *GatewayQueries) CountIntentsForBase(projectSlug, external, method, base string) (int64, error) {
	rows, err := q.intentsForMethod(projectSlug, external, method)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, r := range rows {
		if intentBelongsToBase(r.OrderID, base) {
			n++
		}
	}
	return n, nil
}

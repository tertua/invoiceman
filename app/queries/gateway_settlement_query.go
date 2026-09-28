package queries

import "github.com/tertua/tupay/app/models"

// SettlementStatusRow aggregates one transaction status for the admin settlement summary.
type SettlementStatusRow struct {
	Status    string `gorm:"column:status" json:"status"`
	Count     int64  `gorm:"column:count" json:"count"`
	AmountIDR int64  `gorm:"column:amount_idr" json:"amount_idr"`
}

// SettlementSummary groups every gateway transaction by status with a count and the summed IDR amount (the physical column is amount_id_r — default GORM naming, which ignores the struct's db tag).
func (q *GatewayQueries) SettlementSummary() ([]SettlementStatusRow, error) {
	out := []SettlementStatusRow{}
	err := q.Model(&models.GatewayTransaction{}).
		Select("status, COUNT(*) AS count, COALESCE(SUM(amount_id_r), 0) AS amount_idr").
		Group("status").
		Order("status").
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

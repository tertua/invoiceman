package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// DashboardQueries provides dashboard aggregate queries.
type DashboardQueries struct {
	*gorm.DB
}

// GetStats returns dashboard aggregate numbers for a user.
func (q *DashboardQueries) GetStats(userID uuid.UUID, currency string) (models.DashboardStats, error) {
	stats := models.DashboardStats{}

	var invoices []models.Invoice
	invoiceQuery := q.Select("id, status, due_date, total").Where("user_id = ?", userID)
	if currency != "" {
		invoiceQuery = invoiceQuery.Where("currency = ?", currency)
	}
	if err := invoiceQuery.Find(&invoices).Error; err != nil {
		return stats, err
	}
	stats.InvoiceCount = len(invoices)

	var clientCount int64
	if err := q.Model(&models.Client{}).Where("user_id = ?", userID).Count(&clientCount).Error; err != nil {
		return stats, err
	}
	stats.ClientCount = int(clientCount)

	type paidRow struct {
		InvoiceID uuid.UUID
		Paid      decimal.Decimal
	}
	var paidRows []paidRow
	paidInvoiceQuery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ? AND payments.voided_at IS NULL", userID)
	if currency != "" {
		paidInvoiceQuery = paidInvoiceQuery.Where("invoices.currency = ?", currency)
	}
	if err := paidInvoiceQuery.Group("invoice_id").Scan(&paidRows).Error; err != nil {
		return stats, err
	}
	paidByInvoice := make(map[uuid.UUID]decimal.Decimal, len(paidRows))
	for _, row := range paidRows {
		paidByInvoice[row.InvoiceID] = row.Paid
	}

	var totalRevenue decimal.Decimal
	for _, paid := range paidByInvoice {
		totalRevenue = totalRevenue.Add(paid)
	}
	stats.TotalRevenue = totalRevenue

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var paidThisMonth decimal.Decimal
	paidQuery := q.Model(&models.Payment{}).
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ? AND payments.created_at >= ? AND payments.voided_at IS NULL", userID, monthStart)
	if currency != "" {
		paidQuery = paidQuery.Where("invoices.currency = ?", currency)
	}
	if err := paidQuery.Select("COALESCE(SUM(payments.amount), 0)").Scan(&paidThisMonth).Error; err != nil {
		return stats, err
	}
	stats.PaidThisMonth = paidThisMonth

	pending := pendingInvoiceSet(q.DB, userID)
	for _, invoice := range invoices {
		paid := paidByInvoice[invoice.ID]
		balance := invoice.Total.Sub(paid)
		if !balance.GreaterThan(decimal.Zero) {
			continue
		}
		// Effective status, like reports: a pending overlay still owes even
		// on a flipped-back draft, and pending is never double-counted overdue.
		status := models.ResolveEffectiveStatus(invoice.Status, invoice.DueDate, invoice.Total, paid, pending[invoice.ID])
		if status != models.InvoiceStatusSent && status != models.InvoiceEffectiveOverdue && status != models.InvoiceEffectivePending {
			continue
		}
		stats.Outstanding = stats.Outstanding.Add(balance)
		if status == models.InvoiceEffectiveOverdue {
			stats.OverdueCount++
			stats.OverdueTotal = stats.OverdueTotal.Add(balance)
		}
	}

	return stats, nil
}

// GetRevenueSeries returns revenue per month for the last 6 months.
func (q *DashboardQueries) GetRevenueSeries(userID uuid.UUID, currency string) ([]models.RevenuePoint, error) {
	now := time.Now()
	oldest := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -5, 0)

	type monthlyRow struct {
		CreatedAt time.Time
		Amount    decimal.Decimal
	}
	var rows []monthlyRow
	revenueQuery := q.Model(&models.Payment{}).
		Select("payments.created_at AS created_at, payments.amount AS amount").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ? AND payments.created_at >= ? AND payments.voided_at IS NULL", userID, oldest)
	if currency != "" {
		revenueQuery = revenueQuery.Where("invoices.currency = ?", currency)
	}
	if err := revenueQuery.Scan(&rows).Error; err != nil {
		return nil, err
	}

	revenueByMonth := make(map[string]decimal.Decimal, 6)
	for _, row := range rows {
		key := row.CreatedAt.Format("2006-01")
		revenueByMonth[key] = revenueByMonth[key].Add(row.Amount)
	}

	points := make([]models.RevenuePoint, 0, 6)
	for i := 5; i >= 0; i-- {
		month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -i, 0)
		points = append(points, models.RevenuePoint{
			Key:     month.Format("2006-01"),
			Label:   month.Format("Jan"),
			Revenue: revenueByMonth[month.Format("2006-01")],
		})
	}

	return points, nil
}

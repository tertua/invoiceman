package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"gorm.io/gorm"
)

// DashboardQueries struct for dashboard aggregate queries.
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
		Paid      float64
	}
	var paidRows []paidRow
	paidInvoiceQuery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ?", userID)
	if currency != "" {
		paidInvoiceQuery = paidInvoiceQuery.Where("invoices.currency = ?", currency)
	}
	if err := paidInvoiceQuery.Group("invoice_id").Scan(&paidRows).Error; err != nil {
		return stats, err
	}
	paidByInvoice := make(map[uuid.UUID]float64, len(paidRows))
	for _, row := range paidRows {
		paidByInvoice[row.InvoiceID] = row.Paid
	}

	var totalRevenue float64
	for _, paid := range paidByInvoice {
		totalRevenue += paid
	}
	stats.TotalRevenue = totalRevenue

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var paidThisMonth float64
	paidQuery := q.Model(&models.Payment{}).
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ? AND payments.created_at >= ?", userID, monthStart)
	if currency != "" {
		paidQuery = paidQuery.Where("invoices.currency = ?", currency)
	}
	if err := paidQuery.Select("COALESCE(SUM(payments.amount), 0)").Scan(&paidThisMonth).Error; err != nil {
		return stats, err
	}
	stats.PaidThisMonth = paidThisMonth

	for _, invoice := range invoices {
		// Only sent invoices are receivables; drafts are not billed yet.
		if invoice.Status != models.InvoiceStatusSent {
			continue
		}
		balance := invoice.Total - paidByInvoice[invoice.ID]
		if balance <= 0 {
			continue
		}
		stats.Outstanding += balance
		if invoice.Status == models.InvoiceStatusSent &&
			invoice.DueDate != nil && now.After(*invoice.DueDate) {
			stats.OverdueCount++
			stats.OverdueTotal += balance
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
		Amount    float64
	}
	var rows []monthlyRow
	revenueQuery := q.Model(&models.Payment{}).
		Select("payments.created_at AS created_at, payments.amount AS amount").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ? AND payments.created_at >= ?", userID, oldest)
	if currency != "" {
		revenueQuery = revenueQuery.Where("invoices.currency = ?", currency)
	}
	if err := revenueQuery.Scan(&rows).Error; err != nil {
		return nil, err
	}

	revenueByMonth := make(map[string]float64, 6)
	for _, row := range rows {
		key := row.CreatedAt.Format("2006-01")
		revenueByMonth[key] += row.Amount
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

// recentInvoiceRow is the raw row for recent invoices.
type recentInvoiceRow struct {
	ID            uuid.UUID
	InvoiceNumber string
	ClientName    string
	IssueDate     *time.Time
	Total         float64
	Currency      string
	Status        string
	DueDate       *time.Time
	PaidAmount    float64
	CreatedAt     time.Time
}

// GetRecentInvoices returns the 5 most recent invoices of a user.
func (q *DashboardQueries) GetRecentInvoices(userID uuid.UUID, currency string) ([]models.RecentInvoice, error) {
	invoices := []models.RecentInvoice{}

	paidSubquery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Group("invoice_id")

	var rows []recentInvoiceRow
	recentQuery := q.Table("invoices").
		Select(`invoices.id, invoices.invoice_number,
			COALESCE(clients.name, '') AS client_name,
			invoices.issue_date, invoices.total, invoices.currency,
			invoices.status, invoices.due_date, invoices.created_at,
			COALESCE(pay.paid, 0) AS paid_amount`).
		Joins("LEFT JOIN clients ON clients.id = invoices.client_id").
		Joins("LEFT JOIN (?) AS pay ON pay.invoice_id = invoices.id", paidSubquery).
		Where("invoices.user_id = ?", userID)
	if currency != "" {
		recentQuery = recentQuery.Where("invoices.currency = ?", currency)
	}
	if err := recentQuery.
		Order("invoices.created_at DESC").
		Limit(5).
		Scan(&rows).Error; err != nil {
		return invoices, err
	}

	for _, row := range rows {
		issueDate := ""
		if row.IssueDate != nil {
			issueDate = row.IssueDate.Format(utils.DateLayout)
		}
		invoices = append(invoices, models.RecentInvoice{
			ID:              row.ID.String(),
			InvoiceNumber:   row.InvoiceNumber,
			ClientName:      row.ClientName,
			IssueDate:       issueDate,
			Total:           row.Total,
			Currency:        row.Currency,
			EffectiveStatus: models.ResolveEffectiveStatus(row.Status, row.DueDate, row.Total, row.PaidAmount),
			CreatedAt:       row.CreatedAt,
		})
	}

	return invoices, nil
}

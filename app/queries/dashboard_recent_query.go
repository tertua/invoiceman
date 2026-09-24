package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
)

// recentInvoiceRow is the raw row for recent invoices.
type recentInvoiceRow struct {
	ID            uuid.UUID
	InvoiceNumber string
	ClientName    string
	IssueDate     *time.Time
	Total         decimal.Decimal
	Currency      string
	Status        string
	DueDate       *time.Time
	PaidAmount    decimal.Decimal
	CreatedAt     time.Time
}

// GetRecentInvoices returns the 5 most recent invoices of a user.
func (q *DashboardQueries) GetRecentInvoices(userID uuid.UUID, currency string) ([]models.RecentInvoice, error) {
	invoices := []models.RecentInvoice{}

	paidSubquery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Where("voided_at IS NULL").
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

	pending := pendingInvoiceSet(q.DB, userID)
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
			EffectiveStatus: models.ResolveEffectiveStatus(row.Status, row.DueDate, row.Total, row.PaidAmount, pending[row.ID]),
			CreatedAt:       row.CreatedAt,
		})
	}

	return invoices, nil
}

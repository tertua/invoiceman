package queries

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// ReportQueries provides report aggregation.
type ReportQueries struct {
	*gorm.DB
}

// GetReports aggregates invoices, payments, expenses, and clients for a user.
func (q *ReportQueries) GetReports(userID uuid.UUID, currency string) (models.Reports, error) {
	report := models.Reports{
		Monthly:         make([]models.ReportMonthlyPoint, 0),
		Aging:           make([]models.ReportValuePoint, 0),
		TopClients:      make([]models.ReportClientPoint, 0),
		StatusBreakdown: make([]models.ReportValuePoint, 0),
	}
	var invoices []models.Invoice
	if err := q.Where("user_id = ?", userID).Find(&invoices).Error; err != nil {
		return report, err
	}
	var payments []models.Payment
	if err := q.Where("user_id = ? AND voided_at IS NULL", userID).Find(&payments).Error; err != nil {
		return report, err
	}
	var expenses []models.Expense
	if err := q.Where("user_id = ?", userID).Find(&expenses).Error; err != nil {
		return report, err
	}
	if currency != "" {
		filteredInvoices := make([]models.Invoice, 0, len(invoices))
		for _, invoice := range invoices {
			if invoice.Currency == currency {
				filteredInvoices = append(filteredInvoices, invoice)
			}
		}
		invoices = filteredInvoices
		filteredExpenses := make([]models.Expense, 0, len(expenses))
		for _, expense := range expenses {
			if expense.Currency == currency {
				filteredExpenses = append(filteredExpenses, expense)
			}
		}
		expenses = filteredExpenses
		invoiceIDs := make(map[uuid.UUID]struct{}, len(invoices))
		for _, invoice := range invoices {
			invoiceIDs[invoice.ID] = struct{}{}
		}
		filteredPayments := make([]models.Payment, 0, len(payments))
		for _, payment := range payments {
			if _, ok := invoiceIDs[payment.InvoiceID]; ok {
				filteredPayments = append(filteredPayments, payment)
			}
		}
		payments = filteredPayments
	}
	var clients []models.Client
	if err := q.Where("user_id = ?", userID).Find(&clients).Error; err != nil {
		return report, err
	}

	paidByInvoice := make(map[uuid.UUID]decimal.Decimal)
	for _, payment := range payments {
		report.Totals.Revenue = report.Totals.Revenue.Add(payment.Amount)
		paidByInvoice[payment.InvoiceID] = paidByInvoice[payment.InvoiceID].Add(payment.Amount)
	}
	for _, expense := range expenses {
		report.Totals.Expenses = report.Totals.Expenses.Add(expense.Amount)
	}
	report.Totals.NetProfit = report.Totals.Revenue.Sub(report.Totals.Expenses)

	statusValues := map[string]decimal.Decimal{"draft": decimal.Zero, "sent": decimal.Zero, "overdue": decimal.Zero, "paid": decimal.Zero}
	agingValues := make([]decimal.Decimal, 5)
	clientBilled := make(map[uuid.UUID]decimal.Decimal)
	clientPaid := make(map[uuid.UUID]decimal.Decimal)
	now := time.Now()
	for _, invoice := range invoices {
		paid := paidByInvoice[invoice.ID]
		balance := invoice.Total.Sub(paid)
		status := models.ResolveEffectiveStatus(invoice.Status, invoice.DueDate, invoice.Total, paid)
		// Receivables are billed invoices with an open balance; drafts and
		// paid rows never count toward outstanding/aging.
		if balance.GreaterThan(decimal.Zero) && (status == models.InvoiceStatusSent || status == models.InvoiceEffectiveOverdue) {
			report.Totals.Outstanding = report.Totals.Outstanding.Add(balance)
			bucket := 0
			if invoice.DueDate != nil && now.After(*invoice.DueDate) {
				days := int(now.Sub(*invoice.DueDate).Hours() / 24)
				switch {
				case days <= 30:
					bucket = 1
				case days <= 60:
					bucket = 2
				case days <= 90:
					bucket = 3
				default:
					bucket = 4
				}
			}
			agingValues[bucket] = agingValues[bucket].Add(balance)
		}
		statusValues[status] = statusValues[status].Add(invoice.Total)
		if invoice.ClientID != nil {
			clientBilled[*invoice.ClientID] = clientBilled[*invoice.ClientID].Add(invoice.Total)
			clientPaid[*invoice.ClientID] = clientPaid[*invoice.ClientID].Add(paid)
		}
	}

	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -5, 0)
	monthly := make([]models.ReportMonthlyPoint, 0, 6)
	for i := 0; i < 6; i++ {
		month := start.AddDate(0, i, 0)
		monthly = append(monthly, models.ReportMonthlyPoint{Key: month.Format("2006-01"), Label: month.Format("Jan")})
	}
	monthIndex := func(value time.Time) int {
		return (value.Year()-start.Year())*12 + int(value.Month()-start.Month())
	}
	for _, payment := range payments {
		if payment.PaidOn == nil {
			continue
		}
		index := monthIndex(*payment.PaidOn)
		if index >= 0 && index < len(monthly) {
			monthly[index].Revenue = monthly[index].Revenue.Add(payment.Amount)
		}
	}
	for _, expense := range expenses {
		index := monthIndex(expense.ExpenseDate)
		if index >= 0 && index < len(monthly) {
			monthly[index].Expenses = monthly[index].Expenses.Add(expense.Amount)
		}
	}
	report.Monthly = monthly

	statusNames := map[string]string{"draft": "Draft", "sent": "Sent", "overdue": "Overdue", "paid": "Paid"}
	for _, key := range []string{"draft", "sent", "overdue", "paid"} {
		report.StatusBreakdown = append(report.StatusBreakdown, models.ReportValuePoint{Key: key, Name: statusNames[key], Value: statusValues[key]})
	}
	// Aging buckets use stable keys; the frontend localizes them for display.
	for i, key := range []string{"current", "d1_30", "d31_60", "d61_90", "d90_plus"} {
		report.Aging = append(report.Aging, models.ReportValuePoint{Key: key, Bucket: key, Value: agingValues[i]})
	}

	clientNames := make(map[uuid.UUID]string, len(clients))
	for _, client := range clients {
		clientNames[client.ID] = client.Name
	}
	for id, billed := range clientBilled {
		report.TopClients = append(report.TopClients, models.ReportClientPoint{ID: id.String(), Name: clientNames[id], Billed: billed, Paid: clientPaid[id]})
	}
	sort.Slice(report.TopClients, func(i, j int) bool { return report.TopClients[i].Billed.GreaterThan(report.TopClients[j].Billed) })
	if len(report.TopClients) > 5 {
		report.TopClients = report.TopClients[:5]
	}
	return report, nil
}

package models

// DashboardStats struct to describe dashboard aggregate numbers.
type DashboardStats struct {
	InvoiceCount  int     `db:"invoice_count" json:"invoiceCount"`
	ClientCount   int     `db:"client_count" json:"clientCount"`
	TotalRevenue  float64 `db:"total_revenue" json:"totalRevenue"`
	Outstanding   float64 `db:"outstanding" json:"outstanding"`
	PaidThisMonth float64 `db:"paid_this_month" json:"paidThisMonth"`
	OverdueCount  int     `db:"overdue_count" json:"overdueCount"`
	OverdueTotal  float64 `db:"overdue_total" json:"overdueTotal"`
}

// RevenuePoint struct to describe one dashboard revenue series entry.
type RevenuePoint struct {
	Label   string  `db:"label" json:"label"`
	Revenue float64 `db:"revenue" json:"revenue"`
}

// RecentInvoice struct to describe one dashboard recent invoice entry.
type RecentInvoice struct {
	ID              string  `db:"id" json:"id"`
	InvoiceNumber   string  `db:"invoice_number" json:"invoice_number"`
	ClientName      string  `db:"client_name" json:"client_name"`
	IssueDate       string  `db:"issue_date" json:"issue_date"`
	Total           float64 `db:"total" json:"total"`
	Currency        string  `db:"currency" json:"currency"`
	EffectiveStatus string  `db:"effective_status" json:"effective_status"`
}

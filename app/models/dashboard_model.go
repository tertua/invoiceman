package models

import "time"

// DashboardStats contains dashboard aggregate numbers.
type DashboardStats struct {
	InvoiceCount  int             `db:"invoice_count" json:"invoiceCount"`
	ClientCount   int             `db:"client_count" json:"clientCount"`
	TotalRevenue  Money `db:"total_revenue" json:"totalRevenue"`
	Outstanding   Money `db:"outstanding" json:"outstanding"`
	PaidThisMonth Money `db:"paid_this_month" json:"paidThisMonth"`
	OverdueCount  int    `db:"overdue_count" json:"overdueCount"`
	OverdueTotal  Money `db:"overdue_total" json:"overdueTotal"`
}

// RevenuePoint struct to describe one dashboard revenue series entry.
type RevenuePoint struct {
	Key     string          `db:"key" json:"key,omitempty"`
	Label   string          `db:"label" json:"label"`
	Revenue Money `db:"revenue" json:"revenue"`
}

// RecentInvoice struct to describe one dashboard recent invoice entry.
type RecentInvoice struct {
	ID              string          `db:"id" json:"id"`
	InvoiceNumber   string          `db:"invoice_number" json:"invoice_number"`
	ClientName      string          `db:"client_name" json:"client_name"`
	IssueDate       string          `db:"issue_date" json:"issue_date"`
	Total           Money `db:"total" json:"total"`
	Currency        string          `db:"currency" json:"currency"`
	EffectiveStatus string          `db:"effective_status" json:"effective_status"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
}

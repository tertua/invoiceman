package controllers

import "github.com/tertua/tupay/app/models"

// dashboardResponse is the dashboard payload. Keys keep their historical
// camelCase spelling (revenueSeries, recentInvoices).
type dashboardResponse struct {
	Stats          models.DashboardStats  `json:"stats"`
	RevenueSeries  []models.RevenuePoint  `json:"revenueSeries"`
	RecentInvoices []models.RecentInvoice `json:"recentInvoices"`
}

package controllers

import "github.com/tertua/tupay/app/models"

// reportResponse is the reports payload. Keys keep their historical camelCase
// spelling (topClients, statusBreakdown).
type reportResponse struct {
	Totals          models.ReportTotals         `json:"totals"`
	Monthly         []models.ReportMonthlyPoint `json:"monthly"`
	Aging           []models.ReportValuePoint   `json:"aging"`
	TopClients      []models.ReportClientPoint  `json:"topClients"`
	StatusBreakdown []models.ReportValuePoint   `json:"statusBreakdown"`
}

// newReportResponse lifts the cached reports aggregate into its wire shape.
func newReportResponse(report models.Reports) reportResponse {
	return reportResponse{
		Totals:          report.Totals,
		Monthly:         report.Monthly,
		Aging:           report.Aging,
		TopClients:      report.TopClients,
		StatusBreakdown: report.StatusBreakdown,
	}
}

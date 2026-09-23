package models

import "github.com/shopspring/decimal"

// ReportTotals contains headline totals.
type ReportTotals struct {
	Revenue     decimal.Decimal `json:"revenue"`
	Expenses    decimal.Decimal `json:"expenses"`
	NetProfit   decimal.Decimal `json:"netProfit"`
	Outstanding decimal.Decimal `json:"outstanding"`
}

// ReportMonthlyPoint contains one month of revenue and expenses.
type ReportMonthlyPoint struct {
	Key      string          `json:"key,omitempty"`
	Label    string          `json:"label"`
	Revenue  decimal.Decimal `json:"revenue"`
	Expenses decimal.Decimal `json:"expenses"`
}

// ReportValuePoint contains a labelled report value.
type ReportValuePoint struct {
	Key    string          `json:"key,omitempty"`
	Name   string          `json:"name,omitempty"`
	Bucket string          `json:"bucket,omitempty"`
	Value  decimal.Decimal `json:"value"`
}

// ReportClientPoint contains billing totals for one client.
type ReportClientPoint struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Billed decimal.Decimal `json:"billed"`
	Paid   decimal.Decimal `json:"paid"`
}

// Reports contains the data used by the reports page.
type Reports struct {
	Totals          ReportTotals         `json:"totals"`
	Monthly         []ReportMonthlyPoint `json:"monthly"`
	Aging           []ReportValuePoint   `json:"aging"`
	TopClients      []ReportClientPoint  `json:"topClients"`
	StatusBreakdown []ReportValuePoint   `json:"statusBreakdown"`
}

package models

// ReportTotals contains report headline totals.
type ReportTotals struct {
	Revenue     float64 `json:"revenue"`
	Expenses    float64 `json:"expenses"`
	NetProfit   float64 `json:"netProfit"`
	Outstanding float64 `json:"outstanding"`
}

// ReportMonthlyPoint contains one month of revenue and expenses.
type ReportMonthlyPoint struct {
	Key      string  `json:"key,omitempty"`
	Label    string  `json:"label"`
	Revenue  float64 `json:"revenue"`
	Expenses float64 `json:"expenses"`
}

// ReportValuePoint contains a labelled report value.
type ReportValuePoint struct {
	Key    string  `json:"key,omitempty"`
	Name   string  `json:"name,omitempty"`
	Bucket string  `json:"bucket,omitempty"`
	Value  float64 `json:"value"`
}

// ReportClientPoint contains billing totals for one client.
type ReportClientPoint struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Billed float64 `json:"billed"`
	Paid   float64 `json:"paid"`
}

// Reports contains the data used by the reports page.
type Reports struct {
	Totals          ReportTotals         `json:"totals"`
	Monthly         []ReportMonthlyPoint `json:"monthly"`
	Aging           []ReportValuePoint   `json:"aging"`
	TopClients      []ReportClientPoint  `json:"topClients"`
	StatusBreakdown []ReportValuePoint   `json:"statusBreakdown"`
}

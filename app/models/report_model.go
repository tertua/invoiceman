package models

// ReportTotals contains headline totals.
type ReportTotals struct {
	Revenue     Money `json:"revenue"`
	Expenses    Money `json:"expenses"`
	NetProfit   Money `json:"netProfit"`
	Outstanding Money `json:"outstanding"`
}

// ReportMonthlyPoint contains one month of revenue and expenses.
type ReportMonthlyPoint struct {
	Key      string `json:"key,omitempty"`
	Label    string `json:"label"`
	Revenue  Money  `json:"revenue"`
	Expenses Money  `json:"expenses"`
}

// ReportValuePoint contains a labelled report value.
type ReportValuePoint struct {
	Key    string `json:"key,omitempty"`
	Name   string `json:"name,omitempty"`
	Bucket string `json:"bucket,omitempty"`
	Value  Money  `json:"value"`
}

// ReportClientPoint contains billing totals for one client.
type ReportClientPoint struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Billed Money  `json:"billed"`
	Paid   Money  `json:"paid"`
}

// Reports contains the data used by the reports page.
type Reports struct {
	Totals          ReportTotals         `json:"totals"`
	Monthly         []ReportMonthlyPoint `json:"monthly"`
	Aging           []ReportValuePoint   `json:"aging"`
	TopClients      []ReportClientPoint  `json:"topClients"`
	StatusBreakdown []ReportValuePoint   `json:"statusBreakdown"`
}

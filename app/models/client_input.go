package models

// ClientInput struct to describe create/update client payload.
type ClientInput struct {
	Name    string `json:"name" validate:"required,lte=255"`
	Email   string `json:"email" validate:"omitempty,email,lte=255"`
	Company string `json:"company" validate:"lte=255"`
	Phone   string `json:"phone" validate:"lte=100"`
	Address string `json:"address"`
	Notes   string `json:"notes"`
}

// ClientListRow struct to describe a client row with billing aggregates.
type ClientListRow struct {
	Client
	TotalBilled Money `db:"total_billed" json:"total_billed"`
	Outstanding Money `db:"outstanding" json:"outstanding"`
}

// ClientStats struct to describe client detail statistics.
type ClientStats struct {
	Count       int   `json:"count"`
	TotalBilled Money `json:"totalBilled"`
	Outstanding Money `json:"outstanding"`
}

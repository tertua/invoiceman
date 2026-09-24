package models

// ClientGatewayIdentity is embedded on clients created through a service project.
type ClientGatewayIdentity struct {
	GatewayProjectSlug *string `gorm:"size:64;index:idx_client_gateway_external,unique" db:"gateway_project_slug" json:"-"`
	ExternalID         *string `gorm:"size:128;index:idx_client_gateway_external,unique" db:"external_id" json:"external_id,omitempty"`
}

// InvoiceGatewayIdentity is embedded on invoices created through a service project.
type InvoiceGatewayIdentity struct {
	GatewayProjectSlug *string `gorm:"size:64;index:idx_invoice_gateway_external,unique" db:"gateway_project_slug" json:"-"`
	ExternalID         *string `gorm:"size:128;index:idx_invoice_gateway_external,unique" db:"external_id" json:"external_id,omitempty"`
}

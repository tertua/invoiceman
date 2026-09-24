package queries

import "github.com/tertua/invoiceman/app/models"

// GetGatewayClient returns a project-owned client by its external identity.
func (q *ClientQueries) GetGatewayClient(project, external string) (models.Client, error) {
	client := models.Client{}
	if err := q.Where("gateway_project_slug = ? AND external_id = ?", project, external).First(&client).Error; err != nil {
		return client, notFound(err)
	}
	return client, nil
}

// GetGatewayInvoice returns a project-owned invoice by external identity.
func (q *InvoiceQueries) GetGatewayInvoice(project, external string) (models.Invoice, error) {
	invoice := models.Invoice{}
	if err := q.Where("gateway_project_slug = ? AND external_id = ?", project, external).First(&invoice).Error; err != nil {
		return invoice, notFound(err)
	}
	return invoice, nil
}

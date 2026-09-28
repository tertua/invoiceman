package queries

import "github.com/tertua/tupay/app/models"

// GetGatewayInvoice returns a project-owned invoice by external identity.
func (q *InvoiceQueries) GetGatewayInvoice(project, external string) (models.Invoice, error) {
	invoice := models.Invoice{}
	if err := q.Where("gateway_project_slug = ? AND external_id = ?", project, external).First(&invoice).Error; err != nil {
		return invoice, notFound(err)
	}
	return invoice, nil
}

# Dependency Management

## Rules
- Go dependencies in go.mod, frontend in webui/package.json
- govulncheck ./... after any go.mod/go.sum changes
- Axios import restricted to webui/src/api/http.js (check:bundles:strict CI)
- @react-pdf/renderer restricted to InvoicePdfDownloadContent.jsx (check:bundles:strict CI)
- recharts restricted to Dashboard/Client/Reports chart components (check:bundles:strict CI)
- @ alias for webui/src paths (configured in Vite)
- No new major dependencies without explicit approval

## Examples

### Approved Gateway Libraries
```go
// main.go:96-97
gateway.Register(midtrans.New())
gateway.Register(nowpayments.New())
```

### Query with GORM
```go
// app/queries/invoice_query.go:27-45
func (q *InvoiceQueries) ListInvoices(userID uuid.UUID, status, search, sort, order string, limit, offset int) ([]models.InvoiceListRow, error) {
	invoices := []models.InvoiceListRow{}

	tx := q.filteredInvoices(userID, status, search)

	sortColumn := "invoices.created_at"
	if column, ok := invoiceSortColumns[strings.ToLower(sort)]; ok {
		sortColumn = column
	}
	sortOrder := "DESC"
	if strings.EqualFold(order, "asc") {
		sortOrder = "ASC"
	}
	tx = tx.Order(sortColumn + " " + sortOrder).Order("invoices.created_at DESC")

	if err := tx.Limit(limit).Offset(offset).Scan(&invoices).Error; err != nil {
		return invoices, err
	}

	return invoices, nil
}
```

### Frontend TanStack Query
```js
// webui/src/hooks/useInvoices.js:1-2
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { invoicesApi } from "@/api/invoices";
```

## Anti-patterns

### ❌ Unrestricted Axios Import
```js
// BAD: axios outside http.js (CI fails)
import axios from "axios";
```

### ❌ Recharts Outside Approved Components
```jsx
// BAD: recharts in any component (CI fails check:bundles:strict)
import { PieChart } from "recharts";
```

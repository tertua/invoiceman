package midtrans

import "encoding/json"

func snapPayload(orderID string, amountIDR int64, customer map[string]string, methods []string) ([]byte, error) {
	payload := map[string]any{"transaction_details": map[string]any{"order_id": orderID, "gross_amount": amountIDR}, "customer_details": customer, "item_details": []map[string]any{{"id": orderID, "price": amountIDR, "quantity": 1, "name": "Payment " + orderID}}}
	if len(methods) > 0 {
		payload["enabled_payments"] = methods
	}
	return json.Marshal(payload)
}

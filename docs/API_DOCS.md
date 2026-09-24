# ./docs

Generated Swagger/OpenAPI files live in this directory. Use `/swagger/index.html` for the interactive API reference.

Service integrations authenticate with a project API key and can create an invoice, including its customer, in one request:

```http
POST /api/v1/gateway/invoices
X-Api-Key: <project-api-key>
Idempotency-Key: <unique-request-key>
```

The request uses project-owned `external_id` values for both the invoice and customer. Invoice status may be `draft` or `sent`; `paid` must be produced by a payment settlement flow. Duplicate external invoice IDs return `409`, while retries with the same idempotency key replay the original response.

Payment intents accept an optional provider-neutral `payment_method` such as `qris`, `bank_transfer`, `gopay`, or `crypto`. Use `GET /api/v1/gateway/methods` to discover available methods. Downstream projects do not send Midtrans or NOWPayments provider names; Invoiceman routes the method to the configured provider.

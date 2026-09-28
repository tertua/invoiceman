# ./docs

Generated Swagger/OpenAPI files live in this directory. Use `/swagger/index.html` for the interactive API reference.

Service integrations authenticate with a project API key and can create an invoice, including its customer, in one request:

```http
POST /api/v1/gateway/invoices
X-Api-Key: <project-api-key>
Idempotency-Key: <unique-request-key>
```

The request uses project-owned `external_id` values for both the invoice and customer. Invoice status may be `draft` or `sent`; `paid` must be produced by a payment settlement flow. Duplicate external invoice IDs return `409`, while retries with the same idempotency key replay the original response.

Payment intents select a payment method, not a provider. Send an optional
provider-neutral `payment_method` such as `qris`, `bank_transfer`, `gopay`, or
`crypto`; TuPay routes it to whichever configured provider supports it.
`GET /api/v1/gateway/methods` lists the methods available to the project. The
optional `gateway` field is a legacy escape hatch for pinning a specific
provider and should be omitted in new integrations.

Relay webhooks carry both `payment_type` (the provider's raw value, e.g.
Midtrans `bni` or NOWPayments `btc`) and `payment_method` (the neutral id
above), so consumers can migrate without parsing provider-specific strings.
`gross_amount_idr` is an integer number of fiat minor units; fractional crypto
amounts use `amount_decimal` with `currency`.

## Currency conversion

Providers are not required to support every invoice currency (Midtrans is
IDR-only). TuPay converts with a **manual** rate the account owner sets in
`PATCH /api/v1/settings` as `usd_to_idr` (IDR per 1 USD, e.g. `"18000"`); no
realtime FX feed is ever used. A USD intent routed to an IDR-only provider is
charged at that rate, and the transaction records `currency` (charged),
`invoice_currency`/`invoice_amount` (source), and `usd_to_idr` used. When the
rate is unset (or the currency pair is unsupported) the intent is rejected with
`400 currency conversion is not configured` rather than guessing.

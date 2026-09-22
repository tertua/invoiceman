# TODO — Future ideas (postponed until BE/FE foundation is done)

## MCP adapter for AI invoice operations

**Status:** idea only, do not implement yet.

**Goal:** expose Invoiceman's REST API as MCP tools so an AI assistant can
find clients, prepare invoices, inspect invoice status, and create payment
links through structured calls.

**Initial tools:** `list_clients`, `get_client`, `create_invoice`,
`list_invoices`, `get_invoice`, `update_invoice`, and `create_payment_link`.

**Plan (when picked up):**
1. Build an MCP adapter that calls the existing `POST /api/v1/invoices` and
   related REST endpoints instead of duplicating invoice business logic.
2. Use a dedicated service/API-key authentication flow for the adapter, or
   securely propagate a user's session and CSRF token; do not reuse the
   gateway payment API key as general invoice access.
3. Validate tool inputs against the existing invoice/client contracts and
   default AI-created invoices to `draft`.
4. Require explicit confirmation before sending an invoice, creating a
   payment link, sending email, or otherwise causing an external side effect.
5. Add focused route/adapter tests for authorization, client ownership,
   validation, idempotency, and confirmation-sensitive actions.

**Decisions to confirm:** separate MCP server versus a first-party MCP route;
whether MCP access is per-user or per-service project; and the permission
scope for payment and email actions.

## QR code on invoice → public payment link

**Status:** idea only, do not implement yet.

**Goal:** print a QR code on the invoice (HTML preview + PDF) that opens the
invoice's public page (`/pay/:token`) when scanned.

**Why it fits:** no new backend endpoint is needed. `POST /api/payments/online`
is already idempotent per invoice (reuses the existing link,
`app/controllers/payment_controller.go:229`), tokens are permanent with no
expiry (`app/models/payment_link_model.go`), and `/pay/:token` already renders
the invoice with payment options.

**Plan (when picked up):**
1. Add the `qrcode` npm package (Node API, not `qrcode.react`) — it emits a
   PNG data-URL usable in both `<img>` (HTML preview) and `<Image>`
   (`@react-pdf/renderer` PDF).
2. In `InvoiceDetail`, fetch-or-create the public link via the existing
   `paymentsApi.createOnlineLink(invoiceId)`, build the absolute URL with
   `window.location.origin` (backend equivalent: `publicURL()` using
   `APP_PUBLIC_URL`), generate the QR data-URL at runtime, render it in the
   preview.
3. In `InvoicePdfDownloadContent`, add a small QR in the PDF footer using the
   same data-URL. Printed QRs stay valid because tokens never expire.
4. Optional: a "copy link / show QR" button in `InvoiceDetail`.

**Caveats / decisions to confirm:**
- A printed QR exposes the unguessable public link to whoever holds the
  invoice — acceptable only because the link itself is already public by
  design. Confirm this is OK before printing by default.
- Decide scope: (a) QR in preview + PDF only, or (b) plus copy-link/show-QR
  button in `InvoiceDetail`.
- Verify `vite build` bundle budget after adding the dependency
  (`npm run check:bundles`).

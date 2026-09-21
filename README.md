# invoiceman

Project Go baru dari template [tertua/go-template](https://github.com/tertua/go-template).

## Quick start

1. Salin `.env.example` menjadi `.env` dan isi sesuai kebutuhan.
2. Jalankan: `make docker.run`
3. Buka Swagger: http://127.0.0.1:5000/swagger/index.html

## Perintah

`make build` / `make test` / `make web.check` / `make docker.stop`

## Frontend bundle boundaries

The SPA lazy-loads routes and keeps heavy dependencies out of the initial bundle:

- `@react-pdf/renderer` may only be imported by `web/src/components/invoice/InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`.
- `recharts` is reserved for the full chart pages: Dashboard, ClientDetail, and Reports. Small dashboard sparklines use SVG.
- Run `npm --prefix web run check:bundles` for an advisory report or `make web.check` for strict lint, build, and bundle-budget checks.

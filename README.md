# invoiceman

Project Go baru dari template [tertua/go-template](https://github.com/tertua/go-template).

## Quick start

1. Salin `.env.example` menjadi `.env` dan isi sesuai kebutuhan.
2. Jalankan: `make docker.run`
3. Buka Swagger: http://127.0.0.1:5000/swagger/index.html

## Docker

Image hanya berisi binary — secret tidak pernah di-bake. Inject saat run:

`docker run --env-file .env apiserver`

Ada dua mode deploy:

- **Terpisah (default, `Dockerfile`)**: API-only. FE di-build (`make web.check`)
  lalu di-host terpisah (nginx/Cloudflare). Karena FE memanggil `/api/v1`
  secara relatif, host FE harus mem-proxy `/api` dan `/uploads` ke BE —
  kalau beda origin, isi `CORS_ORIGINS` dan pastikan HTTPS (cookie sesi
  `Secure` tidak jalan di HTTP/IP polos). Satu BE bisa melayani banyak FE.
- **Satu container (`Dockerfile.dev`)**: FE di-embed ke image, `/api/*`
  same-origin — tanpa CORS, proxy, atau domain kedua.

  ```bash
  docker build -f Dockerfile.dev -t invoiceman:dev .
  docker run --rm -p 5000:5000 --env-file .env invoiceman:dev
  ```

Compose bisa memakai `image:` + `env_file: .env`; tidak perlu env saat build.

## Turnstile (CAPTCHA)

Opsional, aktif hanya kalau secret diisi. Ada **dua key** berbeda:

| Key | Untuk | Ditaruh di |
|---|---|---|
| Secret key | Verifikasi token di server | Runtime env `TURNSTILE_SECRET` (`.env`) |
| Site key | Render widget di browser | Build-time `VITE_TURNSTILE_SITE_KEY` |

`VITE_*` ditanam Vite ke bundle saat `npm run build`, jadi site key bukan
runtime env. Untuk image embed, lewatkan sebagai build arg:

```bash
docker build -f Dockerfile.dev \
  --build-arg VITE_TURNSTILE_SITE_KEY=<site-key> -t invoiceman:dev .
```

Kedua key harus dari pasangan Cloudflare yang sama. Kalau tidak dipakai,
biarkan kosong: widget tidak dirender dan server melewati verifikasi.

## Perintah

`make build` / `make test` / `make web.check` / `make docker.stop`

## Frontend bundle boundaries

The SPA lazy-loads routes and keeps heavy dependencies out of the initial bundle:

- `@react-pdf/renderer` may only be imported by `web/src/components/invoice/InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`.
- `recharts` is reserved for the full chart pages: Dashboard, ClientDetail, and Reports. Small dashboard sparklines use SVG.
- Run `npm --prefix web run check:bundles` for an advisory report or `make web.check` for strict lint, build, and bundle-budget checks.

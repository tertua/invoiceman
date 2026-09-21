# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.0] - 2026-09-02

First tagged release. Baseline includes the full scaffold: Vite + React 19 +
Tailwind v4 frontend, FastAPI + SQLite backend, cookie-session auth, invoice
CRUD, dashboard, reports, and PDF export.

### Added
- Multi-tenant data isolation: clients, invoices, items, and expenses are
  scoped per user (`user_id`); new registrations get an empty workspace
  (backend `app/models.py`, all routers).
- Full EN/ID internationalization with language switcher in Settings
  (Appearance tab); all pages translated including Landing, Login, Register,
  and PDF invoice export (`src/lib/i18n.js`, `src/context/LangContext.jsx`).
- IDR currency support; `formatMoney` renders IDR without decimals
  (`src/lib/utils.js`).
- Standalone `starter_kit/` (moved to `~/project/starter_kit`): reusable
  Vite + React 19 + Tailwind v4 UI kit with theme, toast, and i18n systems.
- Route-level error pages via `errorElement`.

### Changed
- Resource IDs use full UUIDs instead of short hashes.
- Demo data rebranded to Simata Studio.
- Frontend migrated from mock store to the real FastAPI backend.

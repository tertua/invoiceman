# Security Policy

Thank you for helping keep **TuPay** and its users safe. This document explains
how to report a security vulnerability and which versions receive security
fixes.

## Supported Versions

TuPay follows a single active release channel. Development happens on `dev`,
promoted to `main` (stable), then to `master` (production, tagged `v$(VERSION)`).
Only the latest released version is actively supported for security fixes.

| Version         | Supported          |
| --------------- | ------------------ |
| Latest (`v2.x`) | :white_check_mark: |
| `< v2.0`        | :x:                |

If you are running an older version, please upgrade to the latest release to
receive security updates.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**
TuPay handles transactions and payment data, so we ask that you report privately
so we can fix the issue before it is disclosed.

To report a vulnerability, use one of the following private channels:

- **GitHub private vulnerability reporting** if it is enabled for this
  repository (recommended): *Security* → *Report a vulnerability*.
- **Directly** via [tertua](https://github.com/tertua) — mention in the report
  that it is a security issue.

### What to include

The more context you provide, the faster we can triage:

- The affected version(s)
- A clear description of the issue and its potential impact
- Steps to reproduce, or a minimal proof of concept
- Affected component (e.g. API, web UI, payment gateway relay, auth)
- Any suggested fix, if you have one

### What to expect

- We will acknowledge your report within **48 hours**.
- We will work to confirm, reproduce, and triage the issue, then provide a
  timeline for a fix.
- We will keep you updated on progress. If the issue is confirmed, we will
  coordinate a disclosure date so a fix can ship before details go public.

We follow **responsible disclosure** and ask that you refrain from public
disclosure until we have had a reasonable chance to address the issue.

## Disclosure Policy

- A fix is prepared on `dev` and promoted through `main`/`master` as a release.
- Security advisories (e.g. GitHub advisories) are used to document confirmed
  vulnerabilities once a fix is available.
- We credit reporters who discover and report confirmed issues, if they wish.

## Security Scope

Relevant areas, by architecture:

- **API** (Go + Fiber) — authentication, authorization, and stable error
  handling
- **Web UI** (React + Vite) — XSS, CSRF, and client-side data handling
- **Payment gateway relay** — API-key authentication and provider integration
- **Data layer** — SQL injection, secrets handling, and SQLite/PostgreSQL
  parity

Out of scope: already-published dependencies you run in your own environment
unless a TuPay-specific integration amplifies the exposure.

## Security Best Practices for Users

- Keep secrets (DB credentials, gateway API keys, JWT secrets) in environment
  variables — never commit them to the repository.
- Run the latest released version to receive the newest security fixes.
- Report phishing of your *public pay links* just as you would any other
  abuse.

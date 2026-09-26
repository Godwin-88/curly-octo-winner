# Deployment & Operations Guide

Order of record for shipping Shule360. **Deploy the API first, then the web
app** — the API is backward compatible (bearer tokens *and* cookies are both
accepted), and the web app's same-origin proxy expects the cookie endpoints
added in Phase 2.

> **One-time note (Phase 2 auth migration):** after the cookie-based session
> deploy, existing users are signed out once (their old localStorage tokens
> are no longer used and no cookie exists yet). They simply sign in again;
> sessions then renew automatically while in active use.

---

## 1. Go API (Fly.io, `api/`)

### Required secrets

```bash
fly secrets set \
  DATABASE_URL=... \
  SUPABASE_URL=... SUPABASE_SERVICE_ROLE_KEY=... \
  JWT_SECRET=...   # REQUIRED — the server refuses to boot without it
```

### Strongly recommended before enabling real M-Pesa

| Secret | Why |
| --- | --- |
| `MPESA_ALLOWED_IPS` | CIDR allowlist for Daraja callbacks, e.g. `196.201.214.0/24,...`. **Startup fails in production if M-Pesa credentials are set without it** (config guard). |
| `CORS_ALLOWED_ORIGINS` | Extra CORS origins beyond the defaults (`*.vercel.app`, production frontend, localhost dev). |
| `SENTRY_DSN` | Server-side error tracking. Without it the API runs fine, just reports nowhere. |

### Migrations

Run in numeric order (001 → 034) against the production database before/at
deploy time:

```bash
psql "$DATABASE_URL" -f api/migrations/034_mpesa_idempotency.sql
```

034 adds a unique index on `payments(checkout_request_id)` — required by the
idempotent M-Pesa callback. Verify with:

```sql
SELECT indexname FROM pg_indexes WHERE indexname = 'uq_payments_mpesa_checkout';
```

### Post-deploy checks

```bash
curl https://shule360-api.fly.dev/health    # {"status":"ok","version":"..."}
curl https://shule360-api.fly.dev/metrics   # Prometheus exposition (wire to Grafana/uptime tooling)
```

---

## 2. Web app (Vercel, `web/`)

### Environment variables

| Var | Meaning |
| --- | --- |
| `NEXT_PUBLIC_API_URL` | **Proxy target** for `/api/v1/*` rewrites (e.g. `https://shule360-api.fly.dev`). The browser never calls it directly. A production build without it warns loudly at build time (defaults to `localhost:8080`, which will fail). |
| `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Realtime inbox updates. |
| `NEXT_PUBLIC_SENTRY_DSN` | Client-side error tracking (optional). |
| `SENTRY_DSN` (optional, server) | Server-side Next error tracking. |

### Checks

- Sign in as staff → dashboard renders, sidebar works, `shule360_session`
  cookie is present (HttpOnly) in devtools.
- Parent portal sign-in → `shule360_guardian_session` cookie present.
- All `/api/v1/*` network calls in devtools go to the **same origin**.

---

## 3. CI/observability

- CI (GitHub Actions): gofmt → vet → build → tests (Go); lint → tsc → build →
  npm audit → Playwright smoke + axe (web).
- Metrics: `GET /metrics` exposes request counters + duration histograms with
  bounded cardinality (chi route patterns). Point any Prometheus-compatible
  scraper at it.
- Errors: Sentry (Go + web), both fully DSN-gated no-ops when unset.

---

## 4. Known operational notes

- `api/seed` binary still sits in git history (purge with `git filter-repo` —
  destructive, coordinate before running).
- Staff JWTs are stateless: logout drops the HttpOnly cookie; a copied token
  stays valid until the 24h expiry. Guardian sessions ARE revocable
  (`guardian_sessions` table). Rolling renewal keeps active sessions alive and
  hard-stops sessions after 24h of inactivity.

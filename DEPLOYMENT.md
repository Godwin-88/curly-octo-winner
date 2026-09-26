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

## 1. Go API (Render, `api/`)

`render.yaml` (repo root) is a Render Blueprint: Dashboard → **New →
Blueprint** → pick this repo → **Apply** creates the web service (native Go
runtime, `rootDir: api`, health check on `/health`).

### Required environment variables

The Blueprint prompts for each secret (`sync: false`); set them in the service's
**Environment** tab if you create the service manually instead:

| Var | Notes |
| --- | --- |
| `DATABASE_URL` | Supabase **transaction pooler** URL (port 6543). The API disables pgx's prepared-statement cache automatically for pooler connections. |
| `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY` | Supabase project (staff auth + admin user API). |
| `UPSTASH_REDIS_REST_URL`, `UPSTASH_REDIS_REST_TOKEN` | Upstash Redis REST credentials. |
| `JWT_SECRET` | **REQUIRED** — the server refuses to boot without it. Use a long random value, not the `.env.example` placeholder. |
| `SETTINGS_ENCRYPTION_KEY` | Encrypts the integration credentials each school enters in **Settings → Integrations** (AES-256-GCM). Optional: the API falls back to `JWT_SECRET`, but a dedicated value lets you rotate the session secret without invalidating every school's stored credentials. |
| `PORT` | Already `8080` in the Blueprint (Render's default is 10000; either works as long as the API binds it — it reads `PORT`). |
| `APP_ENV` | `production` (set by the Blueprint; enables the production config guards). |

> **Upstash Redis is not optional in practice:** login rate limiting fails
> *closed* (by design — a Redis outage must not open password/PIN brute
> forcing), so without it every login attempt returns
> `503 RATE_LIMITER_UNAVAILABLE`. The API logs a warning at startup when the
> Redis URL/token are missing.

### Strongly recommended before enabling real M-Pesa

| Secret | Why |
| --- | --- |
| `MPESA_ALLOWED_IPS` | CIDR allowlist for Daraja callbacks, e.g. `196.201.214.0/24,...`. **Startup fails in production if M-Pesa credentials are set without it** (config guard). |
| `CORS_ALLOWED_ORIGINS` | Extra CORS origins beyond the defaults (`*.vercel.app`, production frontend, localhost dev). |
| `SENTRY_DSN` | Server-side error tracking. Without it the API runs fine, just reports nowhere. |

### Migrations

Migrations are plain, forward-only SQL files in `api/migrations/`. They are
applied in filename order by `api/cmd/migrate`, which records each version plus
a checksum of the file in the `schema_migrations` table:

```bash
cd api
go run ./cmd/migrate -status    # applied / pending / drifted — changes nothing
go run ./cmd/migrate            # apply every pending migration
go run ./cmd/migrate -baseline  # mark all as applied WITHOUT running them
```

- **Fresh database:** `make migrate-up` creates the whole schema (001 → 034).
- **Render deploys do not run migrations** (that needs a paid instance's
  pre-deploy command — see `render.yaml`): run `make migrate-up` from a machine
  that can reach the database before deploying API code that depends on new
  columns/tables.
- **Database already built by hand** (e.g. with `psql -f`): run
  `make migrate-baseline` once, then `make migrate-up` for future files.
- Each migration runs in a single transaction *together* with its bookkeeping
  row, so a failing migration leaves the database untouched and the command can
  simply be re-run after the file is fixed.
- `DRIFTED` in `-status` means an already-applied file was edited afterwards;
  the statement is not re-run, but the divergence should be accounted for with
  a new migration.

Demo data for a fresh development/staging environment:

```bash
cd api
make seed-data   # demo tenant + guardians/learners/staff (migrations/seed/seed.sql)
make seed        # Supabase Auth users + guardian PINs for the demo logins
```

034 adds a unique index on `payments(checkout_request_id)` — required by the
idempotent M-Pesa callback. Verify with:

```sql
SELECT indexname FROM pg_indexes WHERE indexname = 'uq_payments_mpesa_checkout';
```

### Post-deploy checks

```bash
# Replace with the service's own onrender.com hostname (Dashboard → Settings).
curl https://shule360-api.onrender.com/health    # {"status":"ok","version":"..."}
curl https://shule360-api.onrender.com/metrics   # Prometheus exposition (wire to Grafana/uptime tooling)
```

### Settings (per-school configuration)

A school principal configures their own school at **Settings** in the admin UI
(`/settings`): school profile, M-Pesa paybill and callback URL, term and
attendance window, feature switches, and integration credentials (M-Pesa,
Africa's Talking, WhatsApp Cloud, Backblaze B2, Groq, Upstash).

- Credentials are **encrypted at rest** (AES-256-GCM, key from
  `SETTINGS_ENCRYPTION_KEY`) and are never returned to a browser; the API only
  reports *which* fields are set. Saving is principal/super_admin only, and every
  change is written to `audit_logs`.
- "Test connection" performs a real API call per provider (Daraja OAuth token,
  Africa's Talking account API, Graph `/me`, B2 authorize, Groq models, Upstash
  PING) and records the outcome, so a principal can confirm credentials before
  relying on them.
- A school can keep using the platform's environment credentials
  (`use_platform_default`) instead of entering its own.
- Runtime consumption of per-school credentials happens through
  `settings.Service.ResolveSecrets`; the boot-time clients still use the
  environment values until each module is switched over.

---

## 2. Web app (Vercel, `web/`)

### Environment variables

| Var | Meaning |
| --- | --- |
| `NEXT_PUBLIC_API_URL` | **Proxy target** for `/api/v1/*` rewrites (e.g. `https://shule360-api.onrender.com`). The browser never calls it directly. A production build without it warns loudly at build time (defaults to `localhost:8080`, which will fail). |
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

# Deployment & Operations Guide

Order of record for shipping Shule360. **Deploy the API first, then the web
app** — the API is backward compatible (bearer tokens *and* cookies are both
accepted), and the web app's same-origin proxy expects the cookie endpoints
added in Phase 2.

From here on that ordering is enforced by `deploy.yml`: the `web` job declares
`needs: api`, so a push to `main` cannot ship the frontend ahead of the API it
proxies to. See **§ 3** for the pipeline and the one-time setup.

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

- **Fresh database:** `make migrate-up` creates the whole schema (001 → 036).
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

## 3. CI/CD — a push to `main` ships both services

Three workflows in `.github/workflows/`:

| Workflow | Trigger | Role |
| --- | --- | --- |
| `ci.yml` | push + PR to `main`/`develop` | Gate only, never deploys: secret scan → Go fmt/vet/build/test → web lint/tsc/build/npm audit → Playwright smoke + axe. |
| `deploy.yml` | **push to `main`** (+ manual) | Release: builds the API, triggers Render, waits for `/health`, then deploys the frontend to Vercel. |
| `migrations.yml` | manual only | Applies forward-only SQL migrations to the production database. |

### 3.1 Release order

`deploy.yml` runs two jobs and **the frontend waits for the API**:

```
push main → CI (must be green) → api job → web job
                              ↳ verify    ↳ verify + vercel pull/build/deploy
                              ↳ POST Render deploy hook
                              ↳ poll GET /health until 200
```

`web` declares `needs: api`, so a broken API build never ships a frontend that
proxies `/api/v1/*` at it. The `/health` poll retries for 5 minutes because a
free Render instance pays a cold-start penalty after spinning down.

### 3.2 One-time GitHub setup

Create an environment named **`production`**
(Repo → Settings → Environments → New environment). Put everything there so
deploy credentials are not reachable from pull-request-triggered workflows.
Optionally add a required reviewer as an extra gate.

**Environment secrets**

| Secret | Where to get it |
| --- | --- |
| `RENDER_DEPLOY_HOOK_URL` | Render → the API service → **Settings → Deploy Hook → Generate deploy hook**. Copy the whole URL. |
| `VERCEL_TOKEN` | Vercel → Account Settings → Tokens → **Generate**. Scope: your team. |
| `VERCEL_ORG_ID` | Vercel → Project Settings → General → *API Reference / IDs*, or the `orgId` in `web/.vercel/project.json`. |
| `VERCEL_PROJECT_ID` | Same place, or the `projectId` in `web/.vercel/project.json`. |
| `DATABASE_URL` | Supabase **transaction pooler** URL (port 6543). Only used by `migrations.yml`. |
| `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY`, `JWT_SECRET` | Only used by `migrations.yml` — `cmd/migrate` calls `config.Load()` before it opens the pool, so all four are required even though only `DATABASE_URL` is read. |

**Environment variables** (not secret)

| Variable | Purpose |
| --- | --- |
| `API_BASE_URL` | The Render origin, e.g. `https://shule360-api.onrender.com`. Enables the post-deploy `/health` check; if unset the step logs a warning and skips. |

### 3.3 Platform setup (once per service)

**Render (API).** Create the service from `render.yaml` (Dashboard → New →
Blueprint → this repo → Apply). `rootDir: api` builds `./cmd/server` on Render's
native Go runtime; the deploy hook created in 3.2 is what `deploy.yml` calls, so
Render rebuilds the current `main` and restarts the service. Copy the production
values from `api/.env` into the service's **Environment** tab — Render is the
only place they should live.

**Vercel (frontend).** The project must have **Root Directory = `web`**
(monorepo). `deploy.yml` uses the `vercel pull` → `vercel build --prebuilt` →
`vercel deploy --prod` flow, so:

- set `NEXT_PUBLIC_API_URL` in the Vercel project's **Production** variables to
  the Render origin. `vercel pull` downloads it, and `next.config.js` bakes it
  into the `/api/v1/*` rewrite target at build time. It is not inlined into
  client JavaScript, so it is not a secret — but it *is* build-time, so a change
  to the API hostname needs a redeploy, not just a restart.
- no application secret is passed through GitHub Actions; the Vercel project
  environment is the single source of truth for the frontend.

### 3.4 Running migrations

Schema changes are **not** automatic — see 3.5. Use the `migrations.yml` workflow
(Repo → Actions → Migrations → Run workflow):

- `confirm=report` (default): runs `cmd/migrate -status` only. Safe, read-only,
  prints applied / pending / drifted.
- `confirm=apply`: applies everything pending, then re-checks `-status` and fails
  the run if anything is still pending.

`dir=migrations/seed` targets the demo data instead.

Run it **after** the migration file is merged and **before** the deploy that
needs the new column. Migrations are forward-only and each runs in one
transaction with its bookkeeping row, so a failure leaves the database untouched
and the workflow can simply be re-run.

### 3.5 Why migrations are not in the deploy pipeline

A migration that half-applies against a production database is far more
expensive than a forgotten one, so `deploy.yml` never runs SQL. Render's
`preDeployCommand` would need a paid instance and would still run migrations
unattended on every deploy. The manual workflow keeps the decision with a human
and is the only place a `production` database is written to.

### 3.6 Handling credentials

`api/.env` holds live Supabase, Upstash, Backblaze, Africa's Talking, Groq and
M-Pesa credentials; `web/.env.local` holds the frontend configuration. Both are
covered by the root `.gitignore` and neither has ever been committed.

- **Never** copy a value from `api/.env` into a workflow file, a `render.yaml`
  entry, or a commit. CI reads deploy credentials from the `production`
  environment; the services read application credentials from Render/Vercel.
- `scripts/secret-scan.sh` is the CI backstop (`ci.yml` → `secret-scan`). It
  fails if a `.env` file is ever tracked, and if any provider key format (Groq
  `gsk_`, Africa's Talking `atsk_`, Supabase `sb_secret_` / service-role JWT,
  Upstash `gAAAA`/`AB0F`/`ABcF` tokens, a `postgres://user:password@` connection
  string) or a secret-shaped `KEY=value` assignment appears in a tracked file.
  Template markers (`your_...`, `change_this`, `${...}`, `=...`) and the dummy
  credentials the Go test suite uses are allowlisted. Run it locally before
  pushing: `./scripts/secret-scan.sh`.
- A key that is committed must be **rotated at the provider**, not just deleted —
  it stays readable in history.
- Two values in the current `api/.env` are still template placeholders and must
  be replaced before that file is used as the source for production secrets:
  `JWT_SECRET` (the example value would let anyone forge a valid token) and
  `MPESA_PASSKEY`. `MPESA_ALLOWED_IPS` must also be set, because `config.Load()`
  refuses to boot in production with M-Pesa credentials and no IP allowlist.

### 3.7 Observability

- Metrics: `GET /metrics` exposes request counters + duration histograms with
  bounded cardinality (chi route patterns). Point any Prometheus-compatible
  scraper at it.
- Errors: Sentry (Go + web), both fully DSN-gated no-ops when unset.

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

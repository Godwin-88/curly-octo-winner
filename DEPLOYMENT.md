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

- **Fresh database:** `make migrate-up` creates the whole schema (001 → 043).
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


## SMS (Africa's Talking)

Sending is `POST /api/v1/messages`. Every recipient is recorded before the
provider is called; an in-process dispatcher sends them, picks up scheduled
messages every 30 seconds, and resumes after a restart without re-sending to
anyone already attempted.

Before a school sends its first live SMS:

| Step | Where |
|---|---|
| Live **username** and **API key** (not `sandbox`), with credit | `AT_USERNAME`, `AT_API_KEY`, or per school under Settings → Integrations |
| Sender name approved for the account, or left empty | `AT_SENDER_ID`, `tenants.at_sender_id`, or the school's integration setting |
| A long random `AT_DLR_TOKEN` | API environment |
| Delivery-report callback `https://<api host>/api/v1/webhooks/sms/dlr/<AT_DLR_TOKEN>` | Africa's Talking dashboard → SMS → Callback URLs |
| Migrations 039, 040 and 041 applied | `make migrate-up` |

Statuses mean exactly this: **sent** — Africa's Talking accepted it;
**delivered** — a delivery report confirmed it; **failed** — refused, or the
report said so. Without the callback, messages stay at *sent*.

`AT_BASE_URL` must never be set in production; the server refuses to start.

## Fees and M-Pesa

Finance bills learners from fee structures, records what is paid and shows
what is owed. The rules the API keeps:

- A learner has one live invoice per term. Billing a grade twice creates
  nothing the second time.
- An invoice's status follows from its payments; it is never set by hand.
- Nothing recorded is deleted. An invoice is voided, a payment is reversed,
  each with who did it and why. The database itself refuses to delete an
  invoice that has a payment.
- Every confirmed payment gets a receipt number (`RCT-2026-00001`), in
  sequence per school and year with no gaps.

**M-Pesa request (STK push).** The request is recorded before Safaricom is
called. It counts as paid only when Safaricom says so: by its callback, or,
when no callback arrives, by the status query the server makes every minute.
A request Safaricom never answers is closed as *outcome unknown* and is never
repeated automatically.

**Paybill payments.** A parent paying from the M-Pesa menu types an account
number: the learner's UPI or the invoice number. The payment goes against that
learner's oldest unpaid invoices. Money that matches nobody, or exceeds what
is owed, waits under Finance → Paybill payments until someone allocates it.

Before a school collects real money:

| Step | Where |
|---|---|
| The school's own Daraja consumer key, secret, passkey and paybill | Settings → Integrations → M-Pesa. Without them the platform's account is used, and the money lands in the platform's paybill |
| The paybill number on the school | Settings → School (`tenants.mpesa_shortcode`); it is how a paybill payment finds its school |
| A long random `MPESA_WEBHOOK_TOKEN` | API environment |
| `MPESA_CALLBACK_URL` = `https://<api host>/api/v1/webhooks/mpesa/<token>/stk` | API environment |
| Confirmation URL `https://<api host>/api/v1/webhooks/mpesa/<token>/c2b/confirmation` and validation URL `…/c2b/validation` registered for the paybill | Safaricom (Daraja "Register URL"); done once per paybill, outside this app |
| `MPESA_ALLOWED_IPS` set to Safaricom's addresses | API environment |
| `MPESA_BASE_URL` = `https://api.safaricom.co.ke` | API environment; production refuses any other host than Safaricom's two |
| Migrations 042 and 043 applied | `make migrate-up` |

Migration 042 adds "one live invoice per learner per term". It stops, changing
nothing, if a learner is already billed twice for a term: void one of the two
invoices and run it again.

Two schools sharing one paybill can only be told apart when the account number
names a learner; a payment that names nobody is logged as an error and not
recorded. Give each school its own paybill.

`{{fee_balance}}` in an SMS is what a parent still owes, from confirmed
payments. It is refused for an audience that includes anyone who owes nothing:
send to "Parents with a fee balance".

## Modules per school

Communications and Finance are switched on or off per school under Platform →
Schools → Choose modules. A module that is off disappears from that school's
menus and its API answers `403 MODULE_NOT_ENABLED` at once; nothing recorded
is deleted. Schools that existed before migration 043 have every module.

## Running everything locally

```bash
docker compose up --build
```

starts Postgres, applies the migrations and demo data, and runs the API
(`localhost:8090`) and the web app (`localhost:3100`). It reads no `.env` file
and reaches no hosted service: a stand-in (`api/cmd/devstub`) plays Supabase
Auth and Africa's Talking, prints each SMS instead of sending it
(`docker compose logs -f devstub`) and posts delivery reports back. Sign-in
details are at the top of `docker-compose.yml`.

The stand-in also plays Safaricom: an M-Pesa request "pays" after three
seconds, and a parent paying the paybill from their phone is

```bash
curl -d 'amount=5000&account=TEST00000003&shortcode=174379' localhost:9191/dev/c2b
```

Browser tests against that stack (run them one file at a time: together they
exceed the sign-in limit of 5 per 15 minutes):

```bash
cd web && E2E_STACK=1 npx playwright test e2e/communications.spec.ts
cd web && E2E_STACK=1 npx playwright test e2e/finance.spec.ts
```

## School context (platform, group, school)

A staff member belongs to one school and always works in it. Two further kinds
of user live in `platform_users`:

- `scope = 'group'` with a `group_id`: may open the schools of that group
  (`tenants.group_id`).
- `scope = 'platform'`: may open any school.

A platform administrator manages all of this in the web app under
**Platform**: *Schools* (which group a school belongs to), *School groups*, and
*Users* (create a platform or group user, reset a password, deactivate). A new
user's password is generated and shown once, on the screen that created it.
The last active platform administrator cannot be deactivated, and an email that
already belongs to a school's staff is refused.

The first platform administrator has nobody to create them, so it is made from
a machine holding the production `DATABASE_URL` and Supabase service key:

```bash
cd api
go run ./cmd/platformuser -email you@example.com -name "Your Name"
```

It prints the generated password once. Everyone after that is created on the
Users screen.

When a platform or group user opens a school, the API gives them a staff row in
that school the first time (`staff.platform_user_id`), so screens that record
"who did this" work for them as for staff. That row is hidden from the school's
staff list and cannot sign in by itself; its role is read from `platform_users`
on every request, and deactivating the user ends their access at once.

They choose the school per request with the `X-School-ID` header; a school
outside their reach answers 404. The web app keeps the choice in the address:
`/w/{group}/{school}/{module}/{section}/{record}`.

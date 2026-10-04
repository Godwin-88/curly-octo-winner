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

M-Pesa is **off by default** and switched on explicitly with `MPESA_ENABLED`
(`render.yaml` sets `false`). While it is off the payment webhook routes are not
mounted at all, and the service does not need `MPESA_ALLOWED_IPS` to boot — so a
service can run on SMS while the paybill is still being set up.

| Secret | Why |
| --- | --- |
| `MPESA_ENABLED` | `true` to switch M-Pesa on. The moment it is on, the rest of this section applies. |
| `MPESA_ALLOWED_IPS` | CIDR allowlist for Daraja callbacks, e.g. `196.201.214.0/24,...`. **Startup fails in production when M-Pesa is enabled without it** (config guard) — `/webhooks/mpesa/stk` accepts an unauthenticated POST, so the allowlist is the only thing stopping a forged payment confirmation. |
| `MPESA_WEBHOOK_TOKEN` | Secret path segment for the tokenised callback routes. Strongly recommended alongside the allowlist. |
| `CORS_ALLOWED_ORIGINS` | Extra CORS origins beyond the defaults (`*.vercel.app`, production frontend, localhost dev). |
| `SENTRY_DSN` | Server-side error tracking. Without it the API runs fine, just reports nowhere. |

> The `your_mpesa_..._here` placeholders from `.env.example` no longer count as
> configured credentials, so leaving them in place will not block a boot. That is
> a safety net for the inferred mode only — set `MPESA_ENABLED` explicitly
> rather than relying on it.

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

- **Fresh database:** `make migrate-up` creates the whole schema (001 → 046).
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

#### Diagnosing "live but every API call returns 502"

Render reports a service **live** from its health check, and it keeps completing
TLS for *any* `*.onrender.com` name — wildcard DNS and a wildcard certificate,
so both of these resolve and present a valid cert whether or not the service
exists:

```bash
getent hosts nonexistent-service.onrender.com   # resolves anyway
```

So neither DNS nor a successful TLS handshake tells you the API is running. The
distinguishing test is whether an **HTTP response** ever arrives:

```bash
curl -m 20 -o /dev/null -w '%{http_code} in %{time_total}s\n' \
  https://shule360-api.onrender.com/health
```

| Result | Meaning |
| --- | --- |
| `200` in a few seconds | Serving normally. |
| `000` after ~20s, no response headers | **Nothing is listening.** Read the Render log — this is a crash-loop, not a cold start. |
| `000` for ~1 min, then `200` | Free plan cold start. Expected after ~15 min idle. |

A cold start eventually succeeds; a crash-loop never does. If the probe hangs,
open the Render log and look for the process exiting immediately after it starts.

#### The crash-loop: `failed to load config`

The most common cause is a variable that the Blueprint prompts for but was never
filled in. The build still succeeds, Render still reports *live*, and then the
binary exits on every start:

```
==> Build successful 🎉
==> Running './server'
ERROR failed to load config error="missing required environment variables: [JWT_SECRET]"
==> Exited with status 1
```

Fix it in the service's **Environment** tab, then **Deploy** again. The variables
that stop a production boot, in the order `config.Load()` checks them:

| Error | Fix |
| --- | --- |
| `missing required environment variables: [DATABASE_URL, SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, JWT_SECRET]` | Set the listed ones. Only `JWT_SECRET` needs a value you invent: `openssl rand -base64 48`. Never paste the `your_jwt_secret_here_change_this_in_production` placeholder — anyone could then forge session tokens signed with it. |
| `MPESA_ALLOWED_IPS must be set in production while M-Pesa is enabled` | M-Pesa is switched on without an IP allowlist. Either set the Safaricom CIDRs, or — if the paybill is not live yet — set `MPESA_ENABLED=false` so the module (and its webhook routes) stay off while SMS runs. |
| `AT_BASE_URL must not be set in production` | Delete it. It is a development switch that would redirect every school's SMS to another host. |
| `MPESA_BASE_URL must be https://api.safaricom.co.ke or https://sandbox.safaricom.co.ke in production` | Use one of those two hosts. `sandbox` is allowed but payments will not be real. |

#### School registration returns 403 `SIGNUP_CLOSED`

A healthy API will still refuse self-registration if `SIGNUP_MODE` is `closed`,
which is what `render.yaml` sets. The response is `403` with
`SIGNUP_CLOSED`: "Schools are set up by the Shule360 team."

To allow schools to register themselves, set `SIGNUP_MODE` in Render to:

- `open` — anyone may register
- `code` — requires a matching `SIGNUP_CODE` of at least 8 characters
- `closed` — nobody; a platform administrator creates schools

Note `SIGNUP_MODE` is read at **boot**, so changing it requires a redeploy.

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

## Academic: observations, registers and report cards

Academic is a module a school has or does not have (Platform → Schools →
Choose modules); it covers the curriculum, observations, attendance and report
cards. Migration 045 gives it to every school that already had an explicit
list of modules.

- **Observations** are recorded by the member of staff who is signed in; only
  that teacher or a principal can remove one.
- **The register** is marked one class at a time and never for a day that has
  not happened (the date is Kenya's). "Text the parents of absent learners" is
  off unless ticked, applies to today's register only, and sends one ordinary
  message per absent learner: it appears under Communications with its
  delivery, respects a parent's opt-out, and is sent once per learner per day
  however often the register is saved. It needs the Communications module.
- **Report cards** are drafts until published. A draft is built from the
  term's observations (the latest level per sub-strand) and can be rebuilt,
  commented on or deleted. A published card is what the parent portal shows; it
  is not edited, rebuilt or deleted until a principal reopens it. The PDF is
  produced when asked for and is not stored; `report_card_pdfs` is no longer
  used.
- **Curriculum.** A new school starts with the seven core competencies and
  eight values. Learning areas are added per grade from Academic → Curriculum
  ("Add the usual learning areas for…"), then edited. **Check that list against
  the current KICD curriculum designs before relying on it**, and note that
  strands and sub-strands are not supplied: the school enters the ones it
  teaches, and observations are recorded against sub-strands.

## Getting a school started

A school and its first administrator are created together, so there is never a
school nobody can sign in to. There are two ways in:

- **The school registers itself** at `/auth/register`: three steps (the school
  and whether it is public or private, the person registering, the current
  term), one request. That person becomes the school's principal and is signed
  in to a checklist of what to set up.
- **A platform administrator creates it** under Platform → Schools → Add
  school, naming the principal. The principal's password is generated and shown
  once, to hand over.

`SIGNUP_MODE` decides whether the first way is available: `open`, `code`
(the form asks for `SIGNUP_CODE`) or `closed`. Unset, it is closed in
production. Registration creates accounts without confirming the email
address, so `open` lets anyone create a school under any unused email: prefer
`code` or `closed` on a live server. Registration shares the sign-in limit of
5 attempts per address per 15 minutes.

Inside a school, a principal manages who can sign in under School setup →
Users: add a person (their password is shown once), change their role, reset a
password, deactivate or reactivate. One email is one person across every
school. A change of role or a deactivation applies at that person's next
request, not when their session expires. A school always keeps one active
principal, and nobody can deactivate themselves.

School setup → Getting started is worked out from what is recorded (users,
learners, learning areas, fee structures and invoices, the paybill, messages),
so it cannot show a step as done that is not.

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

**What a school charges is its own list.** Finance → Fee items holds everything
a school charges for; the school adds, renames and retires entries, and fee
structures are built from the list. Nothing is built in. A new school starts
with the usual items for its kind: a private school with tuition, activity
fee, lunch, transport, boarding and caution money; a public school, which
charges no tuition, with lunch programme, activity, assessment and development
levies, remedial teaching, transport and boarding. Whether a school is public
or private is chosen at registration and changed under Platform → Schools.

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
| Migrations 042 to 044 applied | `make migrate-up` |

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
cd web && E2E_STACK=1 npx playwright test e2e/onboarding.spec.ts
```

## School context (platform, group, school)

A staff member belongs to one school and always works in it. Two further kinds
of user live in `platform_users`:

- `scope = 'group'` with a `group_id`: may open the schools of that group
  (`tenants.group_id`).
- `scope = 'platform'`: may open any school. With the role `super_admin` this
  is the platform's own administrator.

There are exactly two groups, **Public schools** and **Private schools**
(migration 046). Every school is in the one for its kind: a school is created
as public or private, and "Move to public or private" on the school is the
only way it changes group. The database enforces both, so a group cannot be
added or removed and a school cannot be put in the wrong one. A group's name
and description can be edited. A group user therefore opens every public
school, or every private one.

Migration 046 removes any group that existed before. A user who was limited to
one of those is moved to the public or private group and **deactivated**,
because that group holds more schools than they were given; reactivate them
under Platform → Users if that is intended.

A platform administrator manages all of this in the web app under
**Platform**: *Schools* (add a school with its principal, rename it, move it
between public and private, choose its modules), *Groups*, and
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

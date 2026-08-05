# Shule360 — School Management System

A multi-tenant school management platform for Kenyan public schools under the **Competency-Based Curriculum (CBC)**. Shule360 digitizes the full school lifecycle — communications, learner records, academics, transport, and finance — with a focus on the unique needs of Kenyan schools (NEMIS integration, Africa's Talking SMS, WhatsApp Business, CBC assessment rubrics).

> **Status:** EPIC 4 complete — Transport (fleet, routes, trips, live tracking) + Learner Records + Academic + Communications

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Repository Structure](#repository-structure)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Authentication & Multi-Tenancy](#authentication--multi-tenancy)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Testing](#testing)
- [Roadmap](#roadmap)

---

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Backend** | Go 1.22 + Chi router | REST API, business logic |
| **Database** | Supabase (PostgreSQL 15) | Primary data store, RLS, Realtime |
| **Cache / Queue** | Upstash Redis | Rate limiting, job queues, drafts |
| **Vector / Search** | Upstash Vector + Search | Template suggestions, full-text search |
| **Object Storage** | Backblaze B2 | Learner documents, media, evidence |
| **SMS** | Africa's Talking | Bulk SMS, delivery receipts |
| **WhatsApp** | Meta Cloud API v19.0 | Broadcasts, 2-way inbox, chatbot |
| **NEMIS** | Sandbox client (stub) | UPI validation, learner lookup |
| **Frontend** | Next.js 16 (Turbopack) + React 19 + Tailwind | Admin dashboard |
| **Fonts** | Raleway (next/font) | Brand typography |

---

## Architecture

```mermaid
flowchart TB
    subgraph Client["Client Layer"]
        WEB["Next.js 16 Admin Dashboard<br/>(/web)"]
    end

    subgraph API["API Layer (Go /api)"]
        ROUTER["Chi Router<br/>/api/v1"]
        MID["Middleware<br/>JWT Auth · Tenant · Rate Limit"]
        COMMS["Comms Service<br/>SMS · WhatsApp · Inbox"]
        ACAD["Academic Service<br/>Curriculum · Assessments · Attendance"]
        LEARN["Learner Service<br/>Enrollment · Documents · Progression"]
        TRANSP["Transport Service<br/>Vehicles · Routes · Trips · Tracking"]
        NEMIS["NEMIS Client<br/>UPI Validation"]
    end

    subgraph Data["Data Layer"]
        PG[("Supabase Postgres<br/>RLS · Realtime")]
        REDIS[("Upstash Redis<br/>Rate Limit · Queues")]
        VECTOR[("Upstash Vector<br/>Template Suggestions")]
        SEARCH[("Upstash Search<br/>Full-text")]
        B2[("Backblaze B2<br/>Documents")]
    end

    subgraph External["External Providers"]
        AT["Africa's Talking<br/>SMS"]
        META["Meta WhatsApp<br/>Cloud API"]
        NEMISAPI["NEMIS<br/>(Production)"]
    end

    WEB -->|HTTPS / JSON| ROUTER
    ROUTER --> MID
    MID --> COMMS
    MID --> ACAD
    MID --> LEARN
    MID --> TRANSP
    LEARN --> NEMIS
    COMMS --> PG
    ACAD --> PG
    LEARN --> PG
    TRANSP --> PG
    COMMS --> REDIS
    COMMS --> VECTOR
    COMMS --> SEARCH
    LEARN --> B2
    COMMS -->|REST| AT
    COMMS -->|REST| META
    NEMIS -->|REST| NEMISAPI
    META -->|Webhook| ROUTER
    AT -->|DLR Webhook| ROUTER
```

---

## Repository Structure

```mermaid
graph TD
    ROOT["shule360/"] --> API["api/ — Go Backend"]
    ROOT --> WEB["web/ — Next.js Frontend"]
    ROOT --> README["README.md"]
    ROOT --> SPEC["shule360-spec.md"]

    API --> CMD["cmd/server/"]
    API --> INTERNAL["internal/"]
    API --> PKG["pkg/"]
    API --> MIG["migrations/"]
    API --> ENV[".env.example"]
    API --> DOCKER["Dockerfile"]
    API --> FLY["fly.toml"]
    API --> MAKE["Makefile"]

    CMD --> MAIN["main.go — entrypoint, wiring, router"]

    INTERNAL --> ACAD["academic/"]
    INTERNAL --> COMMS["comms/"]
    INTERNAL --> LEARN["learner/"]
    INTERNAL --> MID["middleware/"]
    INTERNAL --> NEMIS["nemis/"]
    INTERNAL --> TRANSP["transport/"]
    INTERNAL --> TENANT["tenant/"]
    INTERNAL --> CONFIG["config/"]

    TRANSP --> TSVC["service.go"]
    TRANSP --> TTRIP["trips.go"]
    TRANSP --> TTYPE["types.go"]
    TRANSP --> THAND["handler.go"]

    ACAD --> ACADH["handler.go"]
    ACAD --> CURR["curriculum/"]
    ACAD --> ASSESS["assessment/"]
    ACAD --> ATT["attendance/"]

    COMMS --> COMMSH["handler.go"]
    COMMS --> SVCS["service.go"]
    COMMS --> SMS["sms/"]
    COMMS --> WA["whatsapp/"]
    COMMS --> INBOX["inbox/"]

    LEARN --> LSVC["service.go"]
    LEARN --> LDOC["documents.go"]
    LEARN --> LPROG["progression.go"]
    LEARN --> LHAND["handler.go"]

    MID --> AUTH["auth.go"]
    MID --> TEN["tenant.go"]
    MID --> RATE["ratelimit.go"]

    PKG --> HTTPUTIL["httputil/"]
    PKG --> SUPABASE["supabase/"]
    PKG --> UPSTASH["upstash/"]
    PKG --> B2["backblaze/"]

    MIG --> M001["001_tenants.sql"]
    MIG --> M002["002_staff_auth.sql"]
    MIG --> M003["003_guardians.sql"]
    MIG --> M004["004_learners.sql"]
    MIG --> M005["005_messages.sql"]
    MIG --> M006["006_message_logs.sql"]
    MIG --> M007["007_wa_conversations.sql"]
    MIG --> M008["008_wa_messages.sql"]
    MIG --> M009["009_curriculum.sql"]
    MIG --> M010["010_assessments.sql"]
    MIG --> M011["011_attendance.sql"]
    MIG --> M012["012_learner_enrollment.sql"]
    MIG --> M013["013_learner_progression.sql"]
    MIG --> M014["014_vehicles.sql"]
    MIG --> M015["015_routes.sql"]
    MIG --> M016["016_trips.sql"]
    MIG --> SEED["seed/seed.sql"]

    WEB --> APP["app/"]
    WEB --> COMP["components/"]
    WEB --> LIB["lib/"]
    WEB --> PKGJSON["package.json"]
    WEB --> NEXT["next.config.js"]
    WEB --> TAIL["tailwind.config.ts"]

    APP --> ADMIN["(admin)/"]
    APP --> AUTH["(auth)/"]
    ADMIN --> DASH["dashboard/"]
    ADMIN --> COMMS["communications/"]
    ADMIN --> ACAD["academic/"]
    ADMIN --> LEARN["learners/"]
    ADMIN --> VEHICLES["vehicles/"]
    ADMIN --> ROUTES["routes/"]
    ADMIN --> TRIPS["trips/"]
    COMP --> LAYOUT["layout/"]
    COMP --> COMMS["comms/"]
    COMP --> UI["ui/"]
    LIB --> API["api.ts — typed fetch wrapper"]
    LIB --> SUPABASE["supabase.ts"]
```

### Detailed File Tree

```
shule360/
├── README.md
├── shule360-spec.md
├── api/                          # Go Backend
│   ├── cmd/server/main.go        # Entrypoint, dependency wiring, router
│   ├── internal/
│   │   ├── academic/             # EPIC 2 — Academic
│   │   │   ├── handler.go        # Curriculum + Assessment + Attendance routes
│   │   │   ├── curriculum/       # KICD learning areas, strands, sub-strands, SLOs
│   │   │   ├── assessment/       # CBC rubric assessments (1-4), term summaries
│   │   │   └── attendance/       # Daily roll call, chronic absenteeism
│   │   ├── comms/                # EPIC 1 — Communications
│   │   │   ├── handler.go        # Message + conversation routes
│   │   │   ├── service.go        # Create/send/list/cancel messages
│   │   │   ├── sms/              # Africa's Talking client + tests
│   │   │   ├── whatsapp/         # Meta Cloud API, chatbot, webhook, templates
│   │   │   └── inbox/            # Conversation inbox service
│   │   ├── learner/              # EPIC 3 — Learner Records
│   │   │   ├── service.go        # CRUD + NEMIS validation
│   │   │   ├── documents.go      # Document upload/list/delete
│   │   │   ├── progression.go    # Promote/retain/transfer
│   │   │   └── handler.go        # 17 learner routes
│   │   ├── transport/            # EPIC 4 — Transport
│   │   │   ├── types.go          # Domain types & requests
│   │   │   ├── service.go        # Vehicles, routes, stops, assignments CRUD
│   │   │   ├── trips.go          # Trips, GPS positions, check-ins
│   │   │   └── handler.go        # 20 transport routes
│   │   ├── middleware/           # JWT auth, tenant, rate limiting
│   │   ├── nemis/                # NEMIS client interface + sandbox stub
│   │   ├── tenant/               # Tenant service
│   │   └── config/               # Env config loader
│   ├── pkg/
│   │   ├── httputil/             # JSON response helpers
│   │   ├── supabase/             # pgx pool + auth client
│   │   ├── upstash/              # Redis, Vector, Search clients
│   │   └── backblaze/            # B2 S3-compatible client
│   ├── migrations/               # 16 SQL migrations + seed
│   ├── .env.example
│   ├── Dockerfile
│   ├── fly.toml
│   └── Makefile
└── web/                          # Next.js 16 Frontend
    ├── app/
    │   ├── (admin)/              # Authenticated admin area
    │   │   ├── dashboard/        # Overview dashboard
    │   │   ├── communications/   # SMS, WhatsApp, Inbox, Templates
    │   │   ├── academic/         # Curriculum, Assessments, Attendance
    │   │   ├── learners/         # List, Register, Detail (EPIC 3)
    │   │   ├── vehicles/         # Fleet management (EPIC 4)
    │   │   ├── routes/           # Routes, stops, assignments (EPIC 4)
    │   │   └── trips/            # Schedule, track, check-ins (EPIC 4)
    │   ├── (auth)/login/         # Login page
    │   ├── layout.tsx            # Root layout (Raleway font)
    │   └── globals.css
    ├── components/
    │   ├── layout/Sidebar.tsx    # Admin navigation
    │   ├── comms/                # Audience builder, thread, stats, composer
    │   └── ui/
    ├── lib/
    │   ├── api.ts                # Typed fetch wrapper (all endpoints)
    │   └── supabase.ts           # Lazy-initialized Supabase client
    ├── next.config.js
    ├── tailwind.config.ts
    └── package.json
```

---

## Database Schema

```mermaid
erDiagram
    TENANTS ||--o{ STAFF : "employs"
    TENANTS ||--o{ GUARDIANS : "has"
    TENANTS ||--o{ LEARNERS : "enrolls"
    TENANTS ||--o{ MESSAGES : "sends"
    TENANTS ||--o{ WA_CONVERSATIONS : "hosts"
    TENANTS ||--o{ LEARNING_AREAS : "uses"
    TENANTS ||--o{ CORE_COMPETENCIES : "defines"
    TENANTS ||--o{ VALUES : "defines"
    TENANTS ||--o{ VEHICLES : "owns"
    TENANTS ||--o{ ROUTES : "operates"

    STAFF ||--o{ MESSAGES : "sends"
    STAFF ||--o{ ATTENDANCE : "marks"
    STAFF ||--o{ ASSESSMENTS : "records"
    STAFF ||--o{ LEARNER_DOCUMENTS : "uploads"
    STAFF ||--o{ LEARNER_PROGRESSIONS : "approves"
    STAFF ||--o{ VEHICLES : "drives"
    STAFF ||--o{ TRIPS : "creates"

    GUARDIANS ||--o{ WA_CONVERSATIONS : "chats"
    GUARDIANS }o--o{ LEARNERS : "guardian_of"

    LEARNERS ||--o{ ATTENDANCE : "has"
    LEARNERS ||--o{ ASSESSMENTS : "receives"
    LEARNERS ||--o{ LEARNER_DOCUMENTS : "owns"
    LEARNERS ||--o{ LEARNER_PROGRESSIONS : "undergoes"
    LEARNERS ||--o{ ROUTE_ASSIGNMENTS : "assigned_to"
    LEARNERS ||--o{ TRIP_CHECKINS : "boards"

    MESSAGES ||--o{ MESSAGE_LOGS : "produces"
    WA_CONVERSATIONS ||--o{ WA_MESSAGES : "contains"

    LEARNING_AREAS ||--o{ STRANDS : "contains"
    STRANDS ||--o{ SUB_STRANDS : "contains"
    SUB_STRANDS ||--o{ LEARNING_OUTCOMES : "defines"
    SUB_STRANDS ||--o{ ASSESSMENTS : "assessed_on"

    VEHICLES ||--o{ ROUTES : "assigned_to"
    ROUTES ||--o{ STOPS : "has"
    ROUTES ||--o{ ROUTE_ASSIGNMENTS : "contains"
    ROUTES ||--o{ TRIPS : "schedules"
    STOPS ||--o{ ROUTE_ASSIGNMENTS : "serves"
    TRIPS ||--o{ TRIP_POSITIONS : "reports"
    TRIPS ||--o{ TRIP_CHECKINS : "tracks"

    TENANTS {
        uuid id PK
        text name
        text school_code
        text phone
        text email
        text address
        text logo_url
        boolean is_active
    }
    STAFF {
        uuid id PK
        uuid tenant_id FK
        text full_name
        text email
        text phone
        text role
        text password_hash
    }
    GUARDIANS {
        uuid id PK
        uuid tenant_id FK
        text full_name
        text phone
        text email
        text relation
    }
    LEARNERS {
        uuid id PK
        uuid tenant_id FK
        text upi
        text full_name
        date date_of_birth
        text grade
        text stream
        text birth_cert_no
        text entry_level
        boolean special_needs
        boolean is_active
        date admission_date
        uuid[] guardian_ids
    }
    MESSAGES {
        uuid id PK
        uuid tenant_id FK
        text channel
        text audience_type
        jsonb audience_filter
        text content
        text status
        timestamp scheduled_at
    }
    MESSAGE_LOGS {
        uuid id PK
        uuid message_id FK
        text phone
        text channel
        text status
        text provider_message_id
    }
    WA_CONVERSATIONS {
        uuid id PK
        uuid tenant_id FK
        uuid guardian_id FK
        text wa_contact_phone
        text status
        int unread_count
    }
    WA_MESSAGES {
        uuid id PK
        uuid conversation_id FK
        text direction
        text content_type
        jsonb content
        text wa_message_id
    }
    LEARNING_AREAS {
        uuid id PK
        uuid tenant_id FK
        text name
        text kicd_code
        text grade_level
    }
    STRANDS {
        uuid id PK
        uuid learning_area_id FK
        text name
        text kicd_code
    }
    SUB_STRANDS {
        uuid id PK
        uuid strand_id FK
        text name
        text kicd_code
    }
    LEARNING_OUTCOMES {
        uuid id PK
        uuid sub_strand_id FK
        text description
        int sort_order
    }
    ASSESSMENTS {
        uuid id PK
        uuid learner_id FK
        uuid sub_strand_id FK
        int rubric_level
        text note
        text[] evidence_urls
        int term
        int year
    }
    ATTENDANCE {
        uuid id PK
        uuid learner_id FK
        date date
        text status
        text reason
        boolean sms_notified
    }
    LEARNER_DOCUMENTS {
        uuid id PK
        uuid learner_id FK
        text doc_type
        text file_name
        text file_url
        text mime_type
        bigint file_size
    }
    LEARNER_PROGRESSIONS {
        uuid id PK
        uuid learner_id FK
        text from_grade
        text to_grade
        text action
        int term
        int year
        text notes
    }
    VEHICLES {
        uuid id PK
        uuid tenant_id FK
        text registration
        text make
        text model
        int capacity
        text status
        date insurance_expiry
        date inspection_expiry
        uuid driver_id
    }
    ROUTES {
        uuid id PK
        uuid tenant_id FK
        text name
        text description
        uuid vehicle_id
        boolean active
    }
    STOPS {
        uuid id PK
        uuid route_id FK
        text name
        int sequence
        float latitude
        float longitude
        text landmark
    }
    ROUTE_ASSIGNMENTS {
        uuid id PK
        uuid route_id FK
        uuid learner_id FK
        uuid stop_id FK
        text direction
    }
    TRIPS {
        uuid id PK
        uuid route_id FK
        uuid vehicle_id FK
        text direction
        text status
        timestamp scheduled_departure
        timestamp actual_departure
        timestamp actual_arrival
        int boarded_count
    }
    TRIP_POSITIONS {
        uuid id PK
        uuid trip_id FK
        float latitude
        float longitude
        real speed_kmh
        real heading_deg
        timestamp reported_at
    }
    TRIP_CHECKINS {
        uuid id PK
        uuid trip_id FK
        uuid learner_id FK
        uuid stop_id FK
        text action
        timestamp checked_at
        boolean sms_notified
    }
```

### Logical Data Flow

```mermaid
sequenceDiagram
    participant Admin as School Admin
    participant Web as Next.js Dashboard
    participant API as Go API
    participant DB as Supabase Postgres
    participant NEMIS as NEMIS Sandbox
    participant AT as Africa's Talking
    participant WA as WhatsApp Cloud

    rect rgb(240, 248, 255)
        Note over Admin,DB: EPIC 3 — Learner Enrollment
        Admin->>Web: Register learner (UPI, name, grade)
        Web->>API: POST /learners
        API->>NEMIS: ValidateUPI(upi)
        NEMIS-->>API: Learner info
        API->>DB: INSERT learners
        DB-->>API: Learner record
        API-->>Web: 201 Created
        Web-->>Admin: Learner registered
    end

    rect rgb(255, 250, 240)
        Note over Admin,DB: EPIC 3 — Progression
        Admin->>Web: Promote learner to Grade 5
        Web->>API: POST /learners/{id}/promote
        API->>DB: INSERT learner_progressions
        API->>DB: UPDATE learners SET grade
        DB-->>API: Progression record
        API-->>Web: 201 Created
    end

    rect rgb(240, 255, 240)
        Note over Admin,DB: EPIC 1 — Bulk SMS
        Admin->>Web: Compose SMS campaign
        Web->>API: POST /messages
        API->>DB: INSERT messages
        API->>AT: Send SMS
        AT-->>API: Provider message ID
        API->>DB: INSERT message_logs
        API-->>Web: Message sent
    end

    rect rgb(255, 240, 255)
        Note over Admin,DB: EPIC 1 — WhatsApp Inbox
        WA-->>API: Webhook (inbound message)
        API->>DB: INSERT wa_messages
        API->>DB: UPDATE wa_conversations
        Admin->>Web: Open inbox
        Web->>API: GET /conversations
        API-->>Web: Conversation list
        Admin->>Web: Send reply
        Web->>API: POST /conversations/{id}/reply
        API->>WA: Send message
    end

    rect rgb(245, 245, 255)
        Note over Admin,DB: EPIC 4 — Trip Tracking
        Admin->>Web: Start trip
        Web->>API: POST /trips/{id}/start
        API->>DB: UPDATE trips SET status=in_progress
        Admin->>Web: Record boarding
        Web->>API: POST /trips/{id}/checkins
        API->>DB: INSERT trip_checkins
        API->>DB: UPDATE trips boarded_count
        Vehicle-->>API: Report GPS position
        API->>DB: INSERT trip_positions
        Admin->>Web: View live position
        Web->>API: GET /trips/{id}
        API-->>Web: Trip + latest position
    end
```

---

## API Endpoints

All endpoints are prefixed with `/api/v1` and require a `Bearer` JWT (except webhooks and `/health`).

### Health & Webhooks

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Liveness check | No |
| POST | `/webhooks/whatsapp` | Meta WhatsApp inbound webhook | No |
| GET | `/webhooks/whatsapp` | Meta webhook verification | No |
| POST | `/webhooks/sms/dlr` | Africa's Talking delivery receipt | No |

### Communications — Messages

| Method | Path | Description |
|--------|------|-------------|
| POST | `/messages` | Create & send a message (SMS/WhatsApp/both) |
| GET | `/messages` | List messages (`?status=&channel=&limit=&offset=`) |
| POST | `/messages/estimate` | Estimate recipient count & cost |
| GET | `/messages/{id}` | Get message + delivery stats |
| GET | `/messages/{id}/logs` | Get per-recipient delivery logs |
| DELETE | `/messages/{id}` | Cancel a scheduled message |

### Communications — Conversations (Inbox)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/conversations` | List conversations (`?status=&assigned_to=`) |
| GET | `/conversations/{id}` | Get conversation + messages |
| POST | `/conversations/{id}/reply` | Send a reply |
| PATCH | `/conversations/{id}/assign` | Assign to staff |
| PATCH | `/conversations/{id}/status` | Update status (open/in_progress/waiting/resolved) |

### Academic — Curriculum

| Method | Path | Description |
|--------|------|-------------|
| GET | `/curriculum/learning-areas` | List learning areas |
| POST | `/curriculum/learning-areas` | Create learning area |
| GET | `/curriculum/learning-areas/{id}` | Get learning area |
| GET | `/curriculum/learning-areas/{id}/strands` | List strands |
| POST | `/curriculum/strands` | Create strand |
| GET | `/curriculum/strands/{id}/sub-strands` | List sub-strands |
| POST | `/curriculum/sub-strands` | Create sub-strand |
| GET | `/curriculum/sub-strands/{id}/learning-outcomes` | List SLOs |
| POST | `/curriculum/learning-outcomes` | Create SLO |
| GET | `/curriculum/core-competencies` | List core competencies |
| POST | `/curriculum/core-competencies` | Create core competency |
| GET | `/curriculum/values` | List values |
| POST | `/curriculum/values` | Create value |

### Academic — Assessments

| Method | Path | Description |
|--------|------|-------------|
| POST | `/assessments` | Create assessment (rubric 1-4) |
| GET | `/assessments/{id}` | Get assessment |
| GET | `/assessments/learner/{learnerId}` | List by learner (`?term=&year=`) |
| GET | `/assessments/term-summary` | Term summaries (`?learner_id=&term=&year=`) |
| DELETE | `/assessments/{id}` | Delete assessment |

### Academic — Attendance

| Method | Path | Description |
|--------|------|-------------|
| POST | `/attendance` | Mark attendance (upsert per learner+date) |
| GET | `/attendance/date` | List by date (`?date=YYYY-MM-DD`) |
| GET | `/attendance/learner/{learnerId}` | List by learner |
| GET | `/attendance/{id}` | Get attendance record |
| DELETE | `/attendance/{id}` | Delete attendance record |

### Learners (EPIC 3)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/learners` | List (`?grade=&stream=&search=&include_inactive=`) |
| POST | `/learners` | Register learner (NEMIS UPI validated) |
| GET | `/learners/{id}` | Get learner |
| PATCH | `/learners/{id}` | Update learner |
| DELETE | `/learners/{id}` | Deactivate (soft delete) |
| POST | `/learners/{id}/reactivate` | Reactivate |
| GET | `/learners/{id}/guardians` | List linked guardians |
| GET | `/learners/{id}/documents` | List documents |
| POST | `/learners/{id}/documents` | Upload document reference |
| GET | `/learners/documents/{docId}` | Get document |
| DELETE | `/learners/documents/{docId}` | Delete document |
| GET | `/learners/{id}/progressions` | List progression history |
| POST | `/learners/{id}/promote` | Promote to next grade |
| POST | `/learners/{id}/retain` | Retain in grade |
| POST | `/learners/{id}/transfer-out` | Transfer out (deactivates) |
| POST | `/learners/{id}/transfer-in` | Transfer in (reactivates) |

### Transport — Vehicles (EPIC 4)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/vehicles` | List vehicles (`?status=`) |
| POST | `/vehicles` | Add vehicle |
| GET | `/vehicles/{id}` | Get vehicle |
| PATCH | `/vehicles/{id}` | Update vehicle |
| DELETE | `/vehicles/{id}` | Delete vehicle |

### Transport — Routes, Stops & Assignments (EPIC 4)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/routes` | List routes (with stops) |
| POST | `/routes` | Create route (+ optional stops) |
| GET | `/routes/{id}` | Get route with stops |
| PATCH | `/routes/{id}` | Update route |
| DELETE | `/routes/{id}` | Delete route (cascades stops/assignments) |
| GET | `/routes/{id}/stops` | List route stops |
| POST | `/routes/{id}/stops` | Add stop |
| DELETE | `/stops/{stopId}` | Delete stop |
| GET | `/routes/{id}/assignments` | List assigned learners |
| POST | `/routes/{id}/assignments` | Assign learner to route |
| DELETE | `/assignments/{assignmentId}` | Remove assignment |

### Transport — Trips & Tracking (EPIC 4)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/trips` | List trips (`?status=&on_date=`) |
| POST | `/trips` | Schedule trip |
| GET | `/trips/{id}` | Get trip + latest position |
| PATCH | `/trips/{id}` | Update trip |
| DELETE | `/trips/{id}` | Delete trip |
| POST | `/trips/{id}/start` | Start trip (in_progress) |
| POST | `/trips/{id}/complete` | Complete trip |
| POST | `/trips/{id}/cancel` | Cancel trip |
| GET | `/trips/{id}/positions` | List GPS history (`?limit=`) |
| POST | `/trips/{id}/positions` | Report GPS position |
| GET | `/trips/{id}/checkins` | List boarded/alighted check-ins |
| POST | `/trips/{id}/checkins` | Record check-in |

---

## Authentication & Multi-Tenancy

- **JWT Auth** — `Authorization: Bearer <token>`; middleware extracts `tenant_id`, `staff_id`, `role` from claims.
- **Tenant Isolation** — Every request sets `app.tenant_id` via `SET LOCAL`; PostgreSQL **Row-Level Security (RLS)** policies on every table enforce `tenant_id = current_setting('app.tenant_id')::uuid`.
- **Rate Limiting** — Upstash Redis-backed sliding window (default 100 req / 10s).

---

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+ (Next.js 16 requires Node 20.9+)
- PostgreSQL 15 (or Supabase project)
- Upstash Redis/Vector/Search
- Backblaze B2 bucket
- Africa's Talking + Meta WhatsApp credentials (for live sending)

### 1. Backend

```bash
cd api
cp .env.example .env          # fill in credentials
make run                      # or: go run ./cmd/server
```

### 2. Frontend

```bash
cd web
cp .env.local.example .env.local
npm install
npm run dev                   # http://localhost:3000
```

### 3. Database Migrations

```bash
# Apply migrations in order (001 → 016), then seed:
psql "$DATABASE_URL" -f migrations/001_tenants.sql
# ... repeat for 002-016 ...
psql "$DATABASE_URL" -f migrations/seed/seed.sql
```

---

## Environment Variables

### Backend (`api/.env`)

| Variable | Description |
|----------|-------------|
| `APP_ENV` | `development` / `production` |
| `PORT` | HTTP port (default `8080`) |
| `DATABASE_URL` | Supabase Postgres connection string |
| `SUPABASE_URL` | Supabase project URL |
| `SUPABASE_SERVICE_ROLE_KEY` | Service role key |
| `JWT_SECRET` | JWT signing secret |
| `UPSTASH_REDIS_URL` / `UPSTASH_REDIS_TOKEN` | Redis |
| `UPSTASH_VECTOR_URL` / `UPSTASH_VECTOR_TOKEN` | Vector |
| `UPSTASH_SEARCH_URL` / `UPSTASH_SEARCH_TOKEN` | Search |
| `B2_ACCOUNT_ID` / `B2_APPLICATION_KEY` / `B2_BUCKET_NAME` / `B2_ENDPOINT` | Backblaze B2 |
| `AT_API_KEY` / `AT_USERNAME` / `AT_SENDER_ID` | Africa's Talking |
| `META_WA_TOKEN` / `META_WA_PHONE_NUMBER_ID` / `META_WA_WEBHOOK_VERIFY_TOKEN` | WhatsApp Cloud API |

### Frontend (`web/.env.local`)

| Variable | Description |
|----------|-------------|
| `NEXT_PUBLIC_API_URL` | Go API base URL (default `http://localhost:8080/api/v1`) |
| `NEXT_PUBLIC_SUPABASE_URL` | Supabase project URL |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Supabase anon key |

---

## Testing

```bash
# Backend
cd api
go test ./...          # unit tests (SMS, NEMIS)
go vet ./...           # static analysis

# Frontend
cd web
npx next build         # production build + type check
```

---

## Roadmap

| Epic | Status | Scope |
|------|--------|-------|
| **EPIC 1** | ✅ Complete | Communications Hub — Bulk SMS, WhatsApp Business, Inbox, Chatbot |
| **EPIC 2** | ✅ Complete | Academic — CBC Curriculum, Formative Assessments, Attendance |
| **EPIC 3** | ✅ Complete | Learner Records — Enrollment, Documents, Progression, Transfers |
| **EPIC 4** | ✅ Complete | Transport — Fleet, Routes, Trips, Live GPS Tracking, Boarding Check-ins |
| **EPIC 5** | ⏳ Planned | Finance — Fees, Invoices, Payments (M-Pesa) |
| **EPIC 6** | ⏳ Planned | Reports & Analytics — Report cards, Dashboards |

> **Note:** This README is updated after every epic to reflect the current architecture, schema, and endpoints.
# Shule360 — School Management System
## Product Specification · New Kenyan Curriculum (CBC)
**Version:** 1.0.0-draft  
**Date:** August 2026  
**Stack:** Go (backend) · Next.js (frontend) · Upstash Redis + Vector + Search · Supabase · Backblaze B2 Gen2  
**Curriculum Alignment:** Kenya's Competency-Based Curriculum (CBC) — KICD Framework

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Capability Canvas Framework](#2-capability-canvas-framework)
3. [Domain Architecture](#3-domain-architecture)
4. [EPIC 1 — Manage School Communications (Priority)](#4-epic-1--manage-school-communications-priority)
5. [EPIC 2 — Manage Academic Operations (CBC)](#5-epic-2--manage-academic-operations-cbc)
6. [EPIC 3 — Manage Learner Records](#6-epic-3--manage-learner-records)
7. [EPIC 4 — Manage School Transport Program](#7-epic-4--manage-school-transport-program)
8. [EPIC 5 — Manage School Finance](#8-epic-5--manage-school-finance)
9. [EPIC 6 — Manage Human Resources](#9-epic-6--manage-human-resources)
10. [EPIC 7 — Manage Supplier & Procurement Operations](#10-epic-7--manage-supplier--procurement-operations)
11. [EPIC 8 — Manage Digital Intelligence (Analytics & AI)](#11-epic-8--manage-digital-intelligence-analytics--ai)
12. [EPIC 9 — Manage Digital Security & Compliance](#12-epic-9--manage-digital-security--compliance)
13. [Technology Architecture](#13-technology-architecture)
14. [Data Models](#14-data-models)
15. [API Contracts](#15-api-contracts)
16. [Infrastructure & Deployment](#16-infrastructure--deployment)
17. [Standards, Regulations & Trends](#17-standards-regulations--trends)
18. [Rollout Phasing](#18-rollout-phasing)
19. [Non-Functional Requirements](#19-non-functional-requirements)
20. [Open Issues & Decision Log](#20-open-issues--decision-log)

---

## 1. Executive Summary

**Shule360** is a production-grade, CBC-aligned school management platform purpose-built for Kenyan primary and secondary institutions. It bridges the administrative, pedagogical, and community layers of school life through a unified digital platform — enabling schools to communicate at scale, manage competency-based learner portfolios, run transport and finance programs, and surface intelligence for data-driven school leadership.

### Problem Statement

Kenyan schools operating under the CBC face compounding operational challenges:

- **Communication fragmentation:** Parent communication is ad-hoc (WhatsApp personal numbers, physical notices), creating inequity, delays, and no audit trail.
- **CBC portfolio management:** The competency-based model requires granular learner performance tracking across strands, sub-strands, and formative assessments — a workflow current general-purpose tools do not support.
- **Transport opacity:** Schools running bus programs lack real-time visibility into vehicle location, boarding confirmations, or fee reconciliation.
- **Financial leakage:** School fee collection, supplier procurement, and petty cash are managed in Excel or paper — creating reconciliation gaps.
- **Data silos:** No single source of truth connects learner records, teacher assessments, fee balances, and parent communications.

### Vision

> A school's operating system — from the classroom to the boardroom to the school gate.

### Design Principles

| Principle | Implementation |
|---|---|
| **CBC-first** | Every academic construct maps to KICD strands/sub-strands/competencies |
| **Mobile-first** | Parents and teachers primarily on feature phones and low-bandwidth Android |
| **Offline-tolerant** | Core teacher flows work offline; sync on reconnect |
| **Privacy-by-design** | GDPR-aligned data handling; Kenya Data Protection Act 2019 compliant |
| **Free-tier-first** | Launch infrastructure selected to operate at zero cloud cost below 500 active users |

---

## 2. Capability Canvas Framework

The capability canvas (sourced from the attached `.cypher` graph model) defines a four-tier hierarchy applied consistently across every domain in this system:

```
Domain
  └─ SubDomain
       └─ Capability
            └─ Epic
                 └─ Feature
                      └─ User Story / Acceptance Criteria
```

### Cross-Cutting Enablers (from canvas ENABLES relationships)

The canvas defines platform-level domains that enable all sector-specific domains. For Shule360, these map as:

| Canvas Enabler Domain | Shule360 Manifestation |
|---|---|
| Manage Digital Core | Platform kernel: auth, tenancy, config |
| Manage Digital IT | Infrastructure: Supabase, Upstash, Backblaze |
| Manage Digital Channels (Non-Commercial) | Communication hub: SMS, WhatsApp, push |
| Manage Digital Intelligence (Non-Commercial) | Analytics engine, CBC progress dashboards |
| Manage Digital Backoffice | Finance, HR, procurement modules |
| Manage Digital Security | RBAC, audit logs, data encryption |
| Manage Digital Inter-Operability & Automation | Webhooks, API gateway, CBC portal integration |

### Governing Standards (canvas `GOVERNED_BY` relationships)

- Kenya Data Protection Act, 2019
- KICD CBC Curriculum Designs (2017–2022)
- CA Kenya (Communications Authority) — Bulk SMS regulations
- META Business Messaging Policy (WhatsApp Business API)
- Kenya National Examinations Council (KNEC) data formats

### Influencing Trends (canvas `INFLUENCED_BY` relationships)

- Mobile-money-first financial inclusion (M-Pesa integration)
- AI-assisted formative assessment feedback
- Low-bandwidth progressive web apps for rural connectivity
- Parent self-service portals reducing admin overhead

---

## 3. Domain Architecture

### Domain Map

```
Shule360
├── Manage School Communications Core Operations       [PRIORITY — Phase 1]
├── Manage Academic (CBC) Core Operations              [Phase 1]
├── Manage Learner Records                             [Phase 1]
├── Manage School Transport Operations                 [Phase 1]
├── Manage School Finance                              [Phase 1]
├── Manage Human Resources                             [Phase 2]
├── Manage Supplier & Procurement Operations           [Phase 2]
└── Manage Digital Intelligence                        [Phase 2]
```

Each domain is governed by a `Standard` node and influenced by a `Trend` node, consistent with the capability canvas schema.

---

## 4. EPIC 1 — Manage School Communications (Priority)

### 4.1 Domain Context

**Domain:** Manage School Communications Core Operations  
**Enabled By:** Manage Digital Channels (Non-Commercial) · Manage Digital Security  
**Standards Governing:** CA Kenya Bulk SMS Guidelines · META WABA Policy · Kenya Data Protection Act 2019  
**Trend Influences:** High SMS penetration in rural Kenya · WhatsApp as primary family communication channel

### 4.2 SubDomains

```
Manage School Communications Core Operations
├── SubDomain: Manage Broadcast Messaging
├── SubDomain: Manage Transport Communication
├── SubDomain: Manage Stakeholder Messaging (Suppliers / Board / Staff)
├── SubDomain: Manage Communication Templates
└── SubDomain: Manage Delivery Intelligence
```

---

### 4.3 Capability: Manage Broadcast Messaging

#### EPIC 1.1 — Bulk SMS Campaigns

**Goal:** Enable the school to send targeted or broadcast SMS messages to parents and guardians using verified sender IDs via a local aggregator (e.g., Africa's Talking, Vonage).

##### Features

**FEAT-1.1.1 — Audience Segment Builder**

Build and save recipient groups from the learner database.

| Attribute | Value |
|---|---|
| Segment types | All parents · Class/grade level · Stream · Fee defaulters · CBC strand cohort · Custom tag |
| Persistence | Segments saved to Supabase; reusable across campaigns |
| Real-time count | Live recipient count shown before send |

**FEAT-1.1.2 — SMS Composition & Scheduling**

| Attribute | Value |
|---|---|
| Character counter | 160 / 320 / 480 SMS units shown live |
| Variable injection | `{{parent_name}}`, `{{learner_name}}`, `{{class}}`, `{{fee_balance}}` |
| Scheduling | Send now · Schedule (datetime picker) · Recurring (weekly/monthly) |
| Draft management | Auto-save drafts to Upstash Redis TTL: 7 days |

**FEAT-1.1.3 — Delivery & Cost Estimation**

Before sending, display:
- Estimated recipient count
- SMS units per message
- Estimated cost (KES) based on aggregator rate
- Confirmation modal with school admin approval required above 500 recipients

**FEAT-1.1.4 — Delivery Reporting**

| Metric | Source |
|---|---|
| Delivered | Aggregator DLR callback → Supabase |
| Failed (invalid number) | Aggregator error codes |
| Opted out | Opt-out registry in Supabase |
| Open rate proxy | Not available for SMS; N/A |

##### User Stories

```
US-1.1.1: As a school administrator, I can select all parents of Grade 4 learners 
          and send them a fee reminder SMS, so that fee collection improves.
          
          Acceptance Criteria:
          - Segment "Grade 4 Parents" returns correct count before send
          - SMS contains parent first name via variable injection
          - DLR status visible within 5 minutes of send
          - Failed numbers flagged for data quality review

US-1.1.2: As a principal, I can schedule a term-opening announcement 2 days 
          in advance, so that I can prepare communications without being 
          on-site at send time.

US-1.1.3: As a finance clerk, I can send fee balance reminders only to parents 
          with outstanding fees, so that I don't annoy parents who have paid.
```

---

#### EPIC 1.2 — WhatsApp Business Messaging

**Goal:** Enable the school to communicate with parents via WhatsApp using the official Meta WhatsApp Business API, supporting rich media messages, two-way conversations, and automated responses.

##### Architecture

```
Next.js Admin UI
      │
      ▼
Go API Gateway
      │
      ├── WhatsApp Cloud API (Meta) ──► Parent WhatsApp
      │         ▲
      │         │ Webhooks (delivery receipts, replies)
      │
      ├── Upstash Redis (conversation state, rate limiting)
      └── Supabase (message logs, opt-in registry, templates)
```

##### Features

**FEAT-1.2.1 — WhatsApp Template Management**

Meta requires pre-approved message templates for outbound messages to non-initiated conversations.

| Template Category | Example Use Cases |
|---|---|
| `UTILITY` | Fee payment confirmation, result release notification |
| `MARKETING` | School events, open days, fundraisers |
| `AUTHENTICATION` | OTP for parent portal login |

Template submission workflow:
1. Admin creates template in Shule360 UI
2. System submits to Meta via Cloud API
3. Approval status polled every 30 min (Upstash cron)
4. Approved templates available for campaigns

**FEAT-1.2.2 — Rich Media Outbound Messages**

Supported payload types for approved templates:

| Type | Use Case | Storage |
|---|---|---|
| Image | Report card thumbnail, event flyers | Backblaze B2 Gen2 |
| Document (PDF) | Full CBC learner report, fee invoice | Backblaze B2 Gen2 |
| Video | School event highlights | Backblaze B2 Gen2 |
| Button (Quick Reply) | "Confirm transport pickup", "Pay now" | Inline in template |
| CTA Button | Link to parent portal, M-Pesa payment | Inline in template |

**FEAT-1.2.3 — Conversation Inbox (Two-way)**

When a parent replies to a WhatsApp message or initiates a conversation, the message enters the school's inbox.

| Feature | Description |
|---|---|
| Inbox assignment | Conversations assigned to staff (class teacher, bursar, admin) |
| Conversation threading | Full thread history per parent contact |
| Internal notes | Staff can add private notes to a conversation |
| Handoff | Transfer conversation to another staff member |
| Auto-close | Conversations auto-closed after 24h of inactivity (META 24h window rule) |
| Status | Open · In Progress · Waiting for Parent · Resolved |

**FEAT-1.2.4 — WhatsApp Chatbot (Automated Flows)**

Rule-based + AI-assisted automation for common parent queries:

| Trigger Phrase | Automated Response |
|---|---|
| "fee balance" / "fees" | Returns learner's current fee statement |
| "results" / "report" | Returns link to latest CBC report card |
| "transport" | Returns today's bus location + ETA |
| "timetable" | Returns class timetable image |
| Unrecognized | "Your message has been received. A staff member will respond shortly." |

AI escalation: Unmatched queries after 3 attempts routed to human inbox.

**FEAT-1.2.5 — Broadcast via WhatsApp**

For parents who have opted in, send campaigns via WhatsApp (richer than SMS).

| Parameter | Value |
|---|---|
| Opt-in requirement | Explicit opt-in stored in Supabase; required by META policy |
| Opt-in capture | During parent portal registration; QR code at school gate |
| Rate limit | META Cloud API: 1,000 conversations/day (free tier) |
| Fallback | If WhatsApp fails, auto-fall back to SMS for same message |

##### User Stories

```
US-1.2.1: As a class teacher, I can send a learner's CBC term report directly 
          to their parent's WhatsApp as a PDF, so that parents receive results 
          without visiting school.

US-1.2.2: As a parent, I can type "fee balance" on WhatsApp and receive my 
          child's current balance instantly, without calling the school.

US-1.2.3: As a school administrator, I can view all open WhatsApp conversations 
          and assign them to the right staff member, so that no parent message 
          is missed.

US-1.2.4: As a bursar, I can send a fee invoice PDF to all fee-defaulting parents 
          via WhatsApp in one action, so that I save time on manual follow-up.
```

---

#### EPIC 1.3 — Transport Communication

**Goal:** Dedicated communication channel between the school, transport parents, and drivers.

##### Features

**FEAT-1.3.1 — Transport Parent Group Messaging**

| Parameter | Value |
|---|---|
| Auto-group | All parents enrolled in transport program added to transport segment |
| Channel | SMS + WhatsApp (dual-channel) |
| Trigger types | Manual broadcast · Schedule-based · GPS-event-triggered |

**FEAT-1.3.2 — Real-Time Trip Alerts (GPS-Triggered)**

Automated SMS/WhatsApp messages triggered by vehicle GPS events:

| GPS Event | Message Sent To | Message Content |
|---|---|---|
| Vehicle departs school | Transport parents | "Bus [REG] has departed school. ETA: [TIME]. Route: [ROUTE]" |
| Vehicle arrives at stop | Parent(s) at that stop | "Bus [REG] is 5 minutes from [STOP_NAME]. Please be ready." |
| Vehicle delayed (>15min) | All transport parents | "Bus [REG] is delayed. New ETA: [TIME]. Reason: [REASON]" |
| Vehicle breakdown | All transport parents + Principal | Emergency notification with driver contact |
| Child boarded | Parent of child | "Your child [NAME] has boarded bus [REG] at [TIME]" |
| Child alighted | Parent of child | "Your child [NAME] alighted at [STOP] at [TIME]" |

**FEAT-1.3.3 — Driver Communication**

| Channel | Use Case |
|---|---|
| In-app (driver PWA) | Real-time instructions, route changes |
| SMS | Backup when app is offline |
| WhatsApp | Media-rich (route maps, updated manifests) |

**FEAT-1.3.4 — Absence & Change Alerts**

Parent can communicate transport changes to school:
- "My child will not take the bus today" → Removes from today's manifest
- "My child needs to be dropped at [alternative stop]" → Alert sent to driver
All changes logged with timestamp and parent identity.

---

#### EPIC 1.4 — Stakeholder Messaging (Suppliers, Board, Staff)

**Goal:** Extend the communication hub beyond parents to all school stakeholders.

##### Features

**FEAT-1.4.1 — Supplier Communication**

| Feature | Description |
|---|---|
| Purchase order SMS | Auto-send PO summary to supplier on approval |
| Delivery confirmation | SMS/WhatsApp to supplier confirming delivery receipt |
| Payment notification | "Your invoice KES [AMOUNT] has been processed" |
| Supplier contact registry | Centralized supplier phone/WhatsApp directory |

**FEAT-1.4.2 — Board of Directors / Management Communication**

| Feature | Description |
|---|---|
| Board meeting notifications | Agenda + venue via WhatsApp document |
| Financial summary reports | Monthly PDF report auto-sent to board members |
| Emergency escalations | Principal can trigger urgent board notification |

**FEAT-1.4.3 — Staff (Internal) Notifications**

| Feature | Description |
|---|---|
| Circular notices | Broadcast to all staff or departments |
| Assignment notifications | "You have been assigned [TASK] due [DATE]" |
| Leave approvals | "Your leave request has been approved/denied" |
| Payslip delivery | Monthly payslip PDF to staff WhatsApp (encrypted) |

---

#### EPIC 1.5 — Communication Templates & Intelligence

**Features:**

**FEAT-1.5.1 — Template Library**
- Pre-built templates for common school communications (fee reminders, CBC result release, school closure, emergency)
- School-branded templates (logo, colors embedded in WhatsApp media messages)
- Template versioning in Supabase

**FEAT-1.5.2 — AI Message Drafting**

Using Upstash Vector (semantic search over past effective messages) + Claude API:
- Suggest message text based on purpose described in natural language
- Tone selector: Formal · Friendly · Urgent
- Swahili / English bilingual drafting

**FEAT-1.5.3 — Delivery Analytics Dashboard**

| Metric | Visualization |
|---|---|
| SMS delivery rate by campaign | Line chart over time |
| WhatsApp open rate (message read receipts) | Bar chart |
| Campaign reach vs. registered parents | Coverage gauge |
| Failed number analysis | Table with data quality flags |
| Cost per message by channel | Cost trend |

**FEAT-1.5.4 — Opt-out & Compliance Management**

| Feature | Description |
|---|---|
| Opt-out registry | Supabase table; respected on every send |
| STOP keyword | SMS "STOP" processed via aggregator webhook |
| WhatsApp block handling | META webhook notifies block; auto-suppress |
| Audit log | Every message sent logged with sender, time, recipient count, content hash |

---

## 5. EPIC 2 — Manage Academic Operations (CBC)

### 5.1 Domain Context

**Domain:** Manage Academic (CBC) Core Operations  
**Governed By:** KICD CBC Curriculum Designs · KNEC Assessment Guidelines  
**Trend:** Shift from norm-referenced grading to competency-based portfolios

### 5.2 SubDomains

```
Manage Academic (CBC) Core Operations
├── SubDomain: Manage Curriculum Structure
├── SubDomain: Manage Timetable & Scheduling
├── SubDomain: Manage Formative Assessment
├── SubDomain: Manage Summative Assessment (KPSEA/KCSE)
└── SubDomain: Manage Learner Portfolio
```

### 5.3 Capabilities & Epics

#### EPIC 2.1 — Manage CBC Curriculum Structure

**Feature: KICD Strand/Sub-strand Catalogue**

The system ships with the complete CBC curriculum structure pre-loaded:

| Level | KICD Concept | System Entity |
|---|---|---|
| Learning Area | e.g., Mathematics, English, Kiswahili | `LearningArea` |
| Strand | e.g., Numbers (Maths) | `Strand` |
| Sub-strand | e.g., Whole Numbers | `SubStrand` |
| Specific Learning Outcomes | SLOs per sub-strand | `LearningOutcome` |
| Core Competency | e.g., Communication, Critical Thinking | `CoreCompetency` |
| Values | e.g., Respect, Responsibility | `Value` |

All curriculum data versioned per academic year in Supabase.

#### EPIC 2.2 — Manage Timetable & Scheduling

| Feature | Description |
|---|---|
| Drag-and-drop timetable builder | Assign learning areas, teachers, rooms per period |
| Conflict detection | Flags double-booking of teachers or rooms |
| Substitute management | Cover teacher assignment with notification |
| Holiday calendar | Kenya public holidays pre-loaded; custom school holidays |
| Timetable export | PDF + WhatsApp-shareable image |

#### EPIC 2.3 — Manage Formative Assessment (CBC)

The CBC requires continuous, portfolio-based assessment. This is the pedagogical core of Shule360.

**FEAT-2.3.1 — Digital Rubric Builder**

Teachers define rubrics aligned to sub-strand learning outcomes:

| Rubric Level | CBC Label | Numeric Equivalent |
|---|---|---|
| Level 1 | Below Expectation | 1 |
| Level 2 | Approaching Expectation | 2 |
| Level 3 | Meeting Expectation | 3 |
| Level 4 | Exceeding Expectation | 4 |

**FEAT-2.3.2 — Observation Recording (Teacher Mobile)**

Teachers record learner observations on a progressive web app (PWA), offline-capable:
- Select learner → Select strand/sub-strand → Select rubric level → Add note
- Attach photo evidence (Backblaze B2 Gen2)
- Voice note transcription (future phase)
- Offline queue synced when connectivity restored

**FEAT-2.3.3 — Learner Portfolio Compilation**

- Aggregates all formative observations per learner per term
- Visualizes strand coverage and performance distribution
- Flags learners with no observations in a strand (teacher alert)
- Parent-facing portfolio view in parent portal

**FEAT-2.3.4 — CBC Report Card Generator**

Generates CBC-compliant report cards:
- Per-strand competency ratings
- Core competency narrative remarks
- Teacher comments per learning area
- Export: PDF (Backblaze B2) + WhatsApp delivery (EPIC 1.2)

---

## 6. EPIC 3 — Manage Learner Records

### 6.1 SubDomains

```
Manage Learner Records
├── SubDomain: Manage Learner Registration & Enrollment
├── SubDomain: Manage Learner Progression (Grade Promotion)
├── SubDomain: Manage Attendance
└── SubDomain: Manage Learner Documents
```

### 6.2 Key Features

#### EPIC 3.1 — Learner Registration & Enrollment

| Field | Description |
|---|---|
| Birth Certificate Number | Required; links to NEMIS |
| UPI (NEMIS ID) | Synced with Kenya NEMIS portal |
| CBC entry point | PP1, PP2, Grade 1–9, Form 1–4 |
| Guardian(s) | Up to 3 guardians with WhatsApp + SMS numbers |
| Special needs | IEP flag; accommodation requirements |
| Photo | Backblaze B2 Gen2 |

#### EPIC 3.2 — Attendance Management

| Feature | Description |
|---|---|
| Morning roll call | Teacher marks attendance on PWA per period or per day |
| SMS alert on absence | Auto-SMS to parent if child absent without prior notice |
| Attendance report | Daily, weekly, monthly attendance per learner |
| Chronic absenteeism flag | Alert to class teacher + admin when attendance < 75% |

#### EPIC 3.3 — Learner Progression

| Feature | Description |
|---|---|
| Grade promotion workflow | End of year batch promotion with exceptions |
| Retention flagging | Learner flagged for retention by class teacher → principal approval |
| Transfer out | Generate transfer letter + complete learner record export |
| Transfer in | Import learner records from previous school |

---

## 7. EPIC 4 — Manage School Transport Program

### 7.1 SubDomains

```
Manage School Transport Operations
├── SubDomain: Manage Vehicle Fleet
├── SubDomain: Manage Routes & Stops
├── SubDomain: Manage Learner Transport Enrollment
├── SubDomain: Manage Real-Time Trip Tracking
└── SubDomain: Manage Transport Fees & Billing
```

### 7.2 Key Features

#### EPIC 4.1 — Fleet Management

| Feature | Description |
|---|---|
| Vehicle registry | Registration, capacity, make/model, insurance expiry |
| Driver profiles | License, PSV certificate, contact, photo |
| Maintenance schedule | Service intervals with SMS reminders to transport manager |
| Insurance/inspection alerts | 30/7/1 day alerts before expiry |

#### EPIC 4.2 — Route & Stop Management

| Feature | Description |
|---|---|
| Route builder | Map-based route with ordered stops |
| Stop GPS coordinates | Used for ETA calculation and triggered alerts |
| Morning / afternoon routes | Separate route configurations |
| Learner ↔ stop mapping | Which stop each learner boards/alights |

#### EPIC 4.3 — Real-Time Trip Tracking

**GPS Integration Architecture:**

```
Vehicle GPS Device (e.g., Teltonika FMB920)
          │ MQTT/HTTP
          ▼
Go Telemetry Service
          │
          ├── Upstash Redis (real-time position cache, TTL 60s)
          ├── Supabase (trip history, geofence events)
          └── Communication Hub (EPIC 1.3 triggered alerts)
```

Parent-facing map (Next.js PWA):
- Live bus location on map
- ETA to their stop
- Boarding confirmation notification

#### EPIC 4.4 — Transport Fees & Billing

| Feature | Description |
|---|---|
| Monthly transport invoice | Auto-generated per enrolled learner |
| M-Pesa integration | STK push for transport fee payment |
| Payment reconciliation | Payments matched to invoices automatically |
| Delinquency management | SMS reminders at 7, 3, 1 day before due date |
| Suspension workflow | Auto-suspend transport access after [N] days overdue |

---

## 8. EPIC 5 — Manage School Finance

### 8.1 SubDomains

```
Manage School Finance
├── SubDomain: Manage Fee Structure
├── SubDomain: Manage Fee Collection & Receipting
├── SubDomain: Manage School Budget
├── SubDomain: Manage Expenditure & Petty Cash
└── SubDomain: Manage Financial Reporting
```

### 8.2 Key Features

#### EPIC 5.1 — Fee Structure Management

| Feature | Description |
|---|---|
| Fee item catalogue | Tuition, caution, transport, activity, boarding (if applicable) |
| Per-grade fee schedules | Different amounts per CBC level |
| Discount/scholarship management | Partial/full fee waivers with approval workflow |
| Sibling discounts | Automatic discount rules for multiple enrolled siblings |

#### EPIC 5.2 — Fee Collection

| Payment Channel | Integration |
|---|---|
| M-Pesa Paybill / Till | Safaricom Daraja API |
| Bank transfer | Manual reconciliation with bank statement import |
| Cash | Cashier receipt entry with auto-printed receipt |
| Cheque | Cheque log with clearance date tracking |

Every payment triggers:
- Receipt SMS to parent
- WhatsApp receipt (if opted in) with PDF receipt attached (Backblaze B2)
- Ledger entry in Supabase

#### EPIC 5.3 — Financial Reporting

| Report | Audience | Frequency |
|---|---|---|
| Daily collection summary | Bursar | Daily |
| Fee defaulters list | Bursar + Principal | Weekly |
| Term income statement | Principal + Board | Per term |
| Annual budget vs. actual | Board | Annual |
| Supplier payment register | Bursar | Monthly |

Reports delivered via:
- In-system PDF viewer
- WhatsApp to board members (EPIC 1.4)
- Email (future phase)

---

## 9. EPIC 6 — Manage Human Resources

### 9.1 SubDomains

```
Manage Human Resources
├── SubDomain: Manage Staff Records
├── SubDomain: Manage Payroll & Benefits
├── SubDomain: Manage Leave & Attendance
└── SubDomain: Manage Staff Performance (TSC-aligned)
```

### 9.2 Key Features

| Feature | Description |
|---|---|
| Staff profiles | TSC number, qualifications, subjects, employment history |
| Payroll computation | Basic salary, allowances, PAYE, NHIF, NSSF deductions |
| Payslip distribution | PDF payslip via WhatsApp (encrypted) monthly |
| Leave management | Application → approval → calendar sync |
| TSC appraisal | Performance forms aligned to TSC teacher appraisal framework |
| Substitute tracking | Cover assignments logged against leave records |

---

## 10. EPIC 7 — Manage Supplier & Procurement Operations

### 10.1 SubDomains

```
Manage Supplier & Procurement Operations
├── SubDomain: Manage Supplier Registry
├── SubDomain: Manage Purchase Requisition & Approval
├── SubDomain: Manage Goods Receipt
└── SubDomain: Manage Supplier Payments
```

### 10.2 Key Features

| Feature | Description |
|---|---|
| Supplier KYC | Business registration, KRA PIN, bank details, contact |
| Requisition workflow | Staff submits → HoD approves → Principal/Bursar approves |
| Purchase order generation | PO with school letterhead; PDF sent to supplier via WhatsApp |
| Goods receipt note (GRN) | Receiving officer confirms delivery; quantity verified |
| Three-way match | PO → GRN → Invoice matched before payment authorisation |
| Supplier communication | All PO/GRN/payment notifications via EPIC 1.4 |

---

## 11. EPIC 8 — Manage Digital Intelligence (Analytics & AI)

### 11.1 SubDomains

```
Manage Digital Intelligence
├── SubDomain: Intelligence Governance (Manage Data Ownership)
├── SubDomain: Academic Analytics
├── SubDomain: Financial Analytics
└── SubDomain: Communication Analytics
```

### 11.2 Key Features

#### EPIC 8.1 — CBC Learning Analytics Dashboard

| Metric | Description |
|---|---|
| Strand coverage heatmap | Which strands have been assessed per class |
| Competency distribution | Percentage of learners at each rubric level per strand |
| At-risk learner radar | ML-flagged learners showing multi-strand underperformance |
| Teacher assessment velocity | Are teachers recording observations on schedule? |
| Class vs. school benchmarks | Internal benchmarking only (no cross-school data sharing) |

#### EPIC 8.2 — Upstash Vector Search (AI Features)

Uses Upstash Vector for semantic capabilities:

| Use Case | Implementation |
|---|---|
| Message template suggestion | Query vector store of past effective messages; return top-3 similar templates |
| Parent query auto-response | Embed incoming WhatsApp query; match against FAQ knowledge base |
| Learner portfolio summarisation | Semantic clustering of teacher observation notes |

---

## 12. EPIC 9 — Manage Digital Security & Compliance

### 12.1 SubDomains

```
Manage Digital Security
├── SubDomain: Manage Identity & Access
├── SubDomain: Manage Data Protection
├── SubDomain: Manage Audit & Compliance
└── SubDomain: Manage Tenancy (Multi-school)
```

### 12.2 Key Features

| Feature | Description |
|---|---|
| RBAC | Roles: Super Admin · Principal · Teacher · Bursar · Transport Manager · Parent · Driver |
| JWT + refresh tokens | Go middleware; tokens in httpOnly cookies |
| Supabase Row Level Security (RLS) | Database-level access control |
| Multi-tenancy | Each school is an isolated tenant; schema-per-tenant in Supabase |
| Data encryption at rest | Supabase encrypted storage; Backblaze B2 server-side encryption |
| Audit log | Every create/update/delete logged with actor, timestamp, IP |
| Kenya Data Protection Act | Data processing register; consent management; right to erasure workflow |
| Parent consent | Explicit consent captured on parent portal registration for WhatsApp opt-in |

---

## 13. Technology Architecture

### 13.1 System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         CLIENTS                              │
│  Next.js Admin PWA  │  Parent Portal PWA  │  Teacher PWA    │
│                     │  (Mobile-first)     │  (Offline PWA)  │
└───────────────┬─────────────────────────────────────────────┘
                │ HTTPS / WebSocket
┌───────────────▼─────────────────────────────────────────────┐
│                    Go API Gateway                            │
│  Chi Router · Middleware (Auth, Rate Limit, Tenant)         │
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │ Comms    │ │ Academic │ │ Finance  │ │ Transport    │  │
│  │ Service  │ │ Service  │ │ Service  │ │ Service      │  │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘  │
└───────┼─────────────┼────────────┼───────────────┼──────────┘
        │             │            │               │
┌───────▼─────────────▼────────────▼───────────────▼──────────┐
│                    DATA LAYER                                 │
│                                                              │
│  Supabase (PostgreSQL + RLS)   Upstash Redis (cache/queue)  │
│  Upstash Vector (embeddings)   Upstash Search (full-text)   │
│  Backblaze B2 Gen2 (files)                                   │
└──────────────────────────────────────────────────────────────┘
        │             │
┌───────▼─────┐ ┌─────▼────────────────────────────────────┐
│  SMS        │ │  WhatsApp Cloud API (Meta)                │
│  Aggregator │ │  GPS Telemetry (MQTT)                     │
│  (AT / SMS  │ │  M-Pesa Daraja API (Safaricom)            │
│   leopard)  │ └──────────────────────────────────────────┘
└─────────────┘
```

### 13.2 Go Backend Structure

```
shule360-api/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── auth/           # JWT, session, RBAC
│   ├── comms/          # SMS, WhatsApp, templates
│   ├── academic/       # CBC curriculum, assessments
│   ├── learner/        # Enrollment, attendance, records
│   ├── transport/      # Fleet, routes, GPS telemetry
│   ├── finance/        # Fees, payments, budgets
│   ├── hr/             # Staff, payroll, leave
│   ├── procurement/    # Suppliers, POs, GRNs
│   ├── intelligence/   # Analytics, vector search
│   └── tenant/         # Multi-school tenancy
├── pkg/
│   ├── upstash/        # Redis, Vector, Search clients
│   ├── supabase/       # Supabase REST/Realtime client
│   ├── backblaze/      # B2 Gen2 S3-compatible client
│   ├── africas_talking/ # SMS aggregator
│   ├── meta_wa/        # WhatsApp Cloud API client
│   └── mpesa/          # Daraja API client
└── migrations/         # SQL migrations (goose)
```

### 13.3 Next.js Frontend Structure

```
shule360-web/
├── app/
│   ├── (admin)/        # School admin layout
│   │   ├── dashboard/
│   │   ├── communications/
│   │   │   ├── sms/
│   │   │   ├── whatsapp/
│   │   │   └── inbox/
│   │   ├── academic/
│   │   ├── learners/
│   │   ├── transport/
│   │   ├── finance/
│   │   ├── hr/
│   │   └── procurement/
│   ├── (parent)/       # Parent portal layout
│   │   ├── dashboard/
│   │   ├── results/
│   │   ├── fees/
│   │   └── transport/
│   └── (teacher)/      # Teacher PWA layout
│       ├── attendance/
│       ├── assessments/
│       └── portfolio/
├── components/
├── lib/
│   ├── api/            # Go backend API client
│   └── realtime/       # Supabase Realtime hooks
└── public/
```

### 13.4 Upstash Usage Map

| Upstash Product | Use Case | TTL / Config |
|---|---|---|
| **Redis** | Session cache, rate limiting, SMS job queue, draft messages, real-time vehicle position cache | Session: 24h · Vehicle: 60s · Drafts: 7d |
| **Vector** | Message template embeddings, parent FAQ knowledge base, learner observation semantic search | Dimension: 1536 (OpenAI ada-002 compatible) |
| **Search** | Full-text search across learner names, supplier names, transaction references | Synced from Supabase on write |

### 13.5 Supabase Schema Overview

```sql
-- Core tenancy
tenants (id, name, logo_url, subscription_tier, created_at)

-- People
learners (id, tenant_id, upi, name, dob, grade, stream, guardian_ids[], ...)
guardians (id, tenant_id, name, phone_primary, phone_wa, email, ...)
staff (id, tenant_id, tsc_number, name, role, phone, ...)
suppliers (id, tenant_id, name, kra_pin, phone_wa, bank_details, ...)

-- Communications
messages (id, tenant_id, channel, audience_type, audience_ids[], content, status, sent_at, ...)
message_logs (id, message_id, recipient_id, status, delivered_at, ...)
wa_conversations (id, tenant_id, guardian_id, status, assigned_to, ...)
wa_messages (id, conversation_id, direction, content_type, content, timestamp)

-- Academic
learning_areas (id, level, name, kicd_code)
strands (id, learning_area_id, name, kicd_code)
sub_strands (id, strand_id, name, slos[])
assessments (id, tenant_id, learner_id, sub_strand_id, rubric_level, note, teacher_id, date)

-- Transport
vehicles (id, tenant_id, reg, capacity, driver_id)
routes (id, tenant_id, name, stops[{name, lat, lng, order}])
transport_enrollments (id, tenant_id, learner_id, route_id, stop_id, term)
trips (id, vehicle_id, route_id, type, started_at, ended_at)
trip_events (id, trip_id, event_type, lat, lng, timestamp)

-- Finance
fee_structures (id, tenant_id, grade, term, items[{name, amount}])
fee_invoices (id, tenant_id, learner_id, term, total, balance)
payments (id, tenant_id, invoice_id, amount, channel, reference, paid_at)
```

### 13.6 Backblaze B2 Gen2 Bucket Layout

```
shule360-{tenant_id}/
├── learner-photos/
├── staff-photos/
├── report-cards/
│   └── {year}/{term}/{learner_id}.pdf
├── receipts/
│   └── {year}/{payment_id}.pdf
├── assessment-evidence/
│   └── {learner_id}/{assessment_id}/
├── supplier-docs/
│   └── {supplier_id}/
└── whatsapp-media/
    └── {message_id}/
```

Access pattern: Pre-signed URLs generated by Go backend; 1-hour expiry for sensitive documents.

---

## 14. Data Models

### 14.1 Message Entity

```go
type Message struct {
    ID           uuid.UUID         `json:"id"`
    TenantID     uuid.UUID         `json:"tenant_id"`
    Channel      MessageChannel    `json:"channel"`       // sms | whatsapp | both
    AudienceType AudienceType      `json:"audience_type"` // all | grade | stream | tag | transport | custom
    AudienceFilter json.RawMessage `json:"audience_filter"`
    ContentType  ContentType       `json:"content_type"`  // text | template | media
    Content      string            `json:"content"`
    TemplateID   *string           `json:"template_id,omitempty"`
    MediaURL     *string           `json:"media_url,omitempty"`
    Status       MessageStatus     `json:"status"`        // draft | scheduled | sending | sent | failed
    ScheduledAt  *time.Time        `json:"scheduled_at,omitempty"`
    SentAt       *time.Time        `json:"sent_at,omitempty"`
    SentBy       uuid.UUID         `json:"sent_by"`
    CreatedAt    time.Time         `json:"created_at"`
}
```

### 14.2 Assessment Observation

```go
type Assessment struct {
    ID           uuid.UUID  `json:"id"`
    TenantID     uuid.UUID  `json:"tenant_id"`
    LearnerID    uuid.UUID  `json:"learner_id"`
    SubStrandID  uuid.UUID  `json:"sub_strand_id"`
    RubricLevel  int        `json:"rubric_level"` // 1-4
    Note         string     `json:"note"`
    EvidenceURLs []string   `json:"evidence_urls"` // Backblaze B2 pre-signed
    TeacherID    uuid.UUID  `json:"teacher_id"`
    AssessedAt   time.Time  `json:"assessed_at"`
    Term         int        `json:"term"` // 1|2|3
    Year         int        `json:"year"`
}
```

---

## 15. API Contracts

### 15.1 Communications API

```
POST   /api/v1/messages                      Create & send/schedule message
GET    /api/v1/messages                      List messages (paginated)
GET    /api/v1/messages/:id                  Get message detail + delivery stats
GET    /api/v1/messages/:id/logs             Get per-recipient delivery log
DELETE /api/v1/messages/:id                  Cancel scheduled message

POST   /api/v1/messages/estimate             Estimate cost & reach before send

GET    /api/v1/conversations                 List WhatsApp conversations
GET    /api/v1/conversations/:id             Get conversation thread
POST   /api/v1/conversations/:id/reply       Reply to conversation
PATCH  /api/v1/conversations/:id/assign      Assign conversation to staff
PATCH  /api/v1/conversations/:id/status      Update status

POST   /api/v1/webhooks/sms                  Aggregator DLR webhook
POST   /api/v1/webhooks/whatsapp             META Cloud API webhook

GET    /api/v1/templates                     List approved WA templates
POST   /api/v1/templates                     Submit new template for META approval
```

### 15.2 Academic API

```
GET    /api/v1/curriculum                    Full CBC curriculum tree
GET    /api/v1/assessments/learner/:id       All assessments for a learner
POST   /api/v1/assessments                   Record new assessment observation
GET    /api/v1/reports/learner/:id/term/:n   Generate/fetch CBC report card PDF

GET    /api/v1/attendance/:class_id/:date    Attendance for a class on a date
POST   /api/v1/attendance                    Bulk mark attendance
```

### 15.3 Transport API

```
GET    /api/v1/transport/vehicles            List fleet
GET    /api/v1/transport/routes              List routes with stops
GET    /api/v1/transport/trips/active        Active trips (real-time)
GET    /api/v1/transport/trips/:id/position  Current GPS position (from Upstash Redis)
POST   /api/v1/transport/trips/:id/board     Mark learner boarded
POST   /api/v1/transport/trips/:id/alight    Mark learner alighted
POST   /api/v1/webhooks/gps                  GPS device telemetry webhook
```

---

## 16. Infrastructure & Deployment

### 16.1 Free-Tier Architecture (Phase 1)

| Service | Free Tier Limits | Usage |
|---|---|---|
| **Supabase** (Free) | 500MB DB, 1GB storage, 2GB bandwidth | Primary DB + Auth + Realtime |
| **Upstash Redis** (Free) | 10,000 commands/day, 256MB | Cache, queues, rate limiting |
| **Upstash Vector** (Free) | 10,000 vectors, 1M queries/month | Template similarity, FAQ matching |
| **Upstash Search** (Free) | 10,000 documents | Learner/supplier full-text search |
| **Backblaze B2 Gen2** (Free) | 10GB storage, 1GB/day download | File storage |
| **Vercel** (Free) | 100GB bandwidth, unlimited deployments | Next.js hosting |
| **Fly.io** (Free) | 3 shared VMs, 160GB outbound | Go API hosting |

**Estimated free-tier headroom:** ~300 active learners, ~600 guardian contacts, ~1,000 SMS/month

### 16.2 Environment Configuration

```bash
# Supabase
SUPABASE_URL=
SUPABASE_ANON_KEY=
SUPABASE_SERVICE_ROLE_KEY=

# Upstash
UPSTASH_REDIS_REST_URL=
UPSTASH_REDIS_REST_TOKEN=
UPSTASH_VECTOR_REST_URL=
UPSTASH_VECTOR_REST_TOKEN=
UPSTASH_SEARCH_REST_URL=
UPSTASH_SEARCH_REST_TOKEN=

# Backblaze B2 Gen2
B2_ACCOUNT_ID=
B2_APPLICATION_KEY=
B2_BUCKET_NAME=
B2_ENDPOINT=                    # S3-compatible endpoint

# SMS Aggregator (Africa's Talking)
AT_API_KEY=
AT_USERNAME=
AT_SENDER_ID=

# WhatsApp Cloud API (Meta)
META_WA_TOKEN=
META_WA_PHONE_NUMBER_ID=
META_WA_WEBHOOK_VERIFY_TOKEN=
META_WA_BUSINESS_ACCOUNT_ID=

# M-Pesa Daraja
MPESA_CONSUMER_KEY=
MPESA_CONSUMER_SECRET=
MPESA_PAYBILL=
MPESA_PASSKEY=
MPESA_CALLBACK_URL=

# App
JWT_SECRET=
ENCRYPTION_KEY=                 # AES-256 for sensitive fields
```

### 16.3 CI/CD Pipeline

```
GitHub → GitHub Actions
    ├── Go: test → build → Docker image → Fly.io deploy
    └── Next.js: lint → test → Vercel deploy (preview on PR, prod on main)
```

---

## 17. Standards, Regulations & Trends

### 17.1 Standards (canvas `GOVERNED_BY`)

| Standard | Impact on Shule360 |
|---|---|
| **Kenya Data Protection Act, 2019** | Consent management, data subject rights, processing register required |
| **KICD CBC Curriculum Designs** | All academic entities must map to official strand/sub-strand/SLO codes |
| **CA Kenya Bulk SMS Guidelines** | Registered sender ID required; opt-out must be honoured within 24h |
| **META WhatsApp Business Policy** | Templates must be pre-approved; 24h conversation window enforced |
| **KNEC Data Formats** | Summative assessment data exportable in KNEC-compatible format |
| **TSC Teacher Service Commission** | Staff records, appraisal forms must align to TSC frameworks |

### 17.2 Trends (canvas `INFLUENCED_BY`)

| Trend | Shule360 Response |
|---|---|
| **M-Pesa dominance** | M-Pesa as primary payment channel; STK push for frictionless payment |
| **WhatsApp as default family comms channel** | WhatsApp-first communication strategy; SMS as fallback only |
| **Low-bandwidth rural connectivity** | PWA with offline-first teacher assessment flows; minimal JS bundles |
| **CBC implementation challenges** | System designed to simplify CBC compliance, not add complexity |
| **AI in education** | Semantic search, observation summarisation, at-risk learner detection |
| **Parent self-service expectations** | Parent portal reducing admin enquiries by target 60% |

---

## 18. Rollout Phasing

### Phase 1 — Foundation (Months 1–3)

**Scope:**
- EPIC 1: Full Communications Hub (SMS + WhatsApp + Inbox)
- EPIC 2: Academic Operations (curriculum structure + formative assessment)
- EPIC 3: Learner Records (enrollment + attendance)
- EPIC 4: Transport Communication (alerts, GPS tracking)
- EPIC 5: Fee Collection (basic)
- EPIC 9: Auth + RBAC + Security

**Target:** 1 pilot school, ≤300 learners

### Phase 2 — Full Operations (Months 4–6)

**Scope:**
- EPIC 5: Full Finance (budget, expenditure, reporting)
- EPIC 6: HR (payroll, leave)
- EPIC 7: Procurement (supplier management, PO workflow)
- EPIC 8: Analytics dashboards

**Target:** 5 schools, ≤1,500 total learners

### Phase 3 — Intelligence & Scale (Months 7–12)

**Scope:**
- AI-powered CBC analytics (at-risk detection, competency prediction)
- Multi-school admin portal
- NEMIS integration
- M-Pesa direct fee collection via parent portal
- WhatsApp chatbot advanced flows

**Target:** 20+ schools; paid subscription tiers introduced

---

## 19. Non-Functional Requirements

| Category | Requirement |
|---|---|
| **Performance** | API p95 response < 300ms (non-file); SMS send initiation < 2s |
| **Availability** | 99.5% uptime during school hours (6am–8pm EAT Mon–Sat) |
| **Offline** | Teacher PWA fully functional offline; sync within 60s of reconnection |
| **Scalability** | Architecture supports horizontal scaling of Go services on Fly.io |
| **Data residency** | Supabase region: af-south-1 (Johannesburg) for KDPA compliance |
| **Accessibility** | WCAG 2.1 AA for parent portal |
| **Language** | UI available in English and Kiswahili; all templates bilingual |
| **Mobile** | Parent portal and teacher PWA optimised for Android 8+ on 3G |
| **Browser support** | Chrome 90+, Firefox 90+, Safari 14+, Samsung Internet 14+ |

---

## 20. Open Issues & Decision Log

| ID | Issue | Owner | Status |
|---|---|---|---|
| OI-001 | SMS aggregator selection: Africa's Talking vs. SMS Leopard vs. Celcom Africa — evaluate pricing per SMS for Kenya networks | Tech Lead | Open |
| OI-002 | META WABA account registration: School needs Facebook Business Manager account — confirm procurement process with pilot school | School Admin | Open |
| OI-003 | GPS device hardware selection for transport tracking — evaluate Teltonika FMB920 vs. Concox GT06 on school budget | Transport Manager | Open |
| OI-004 | NEMIS API access — confirm whether KICD/MoE provides a REST API or CSV export for UPI validation | Tech Lead | Open |
| OI-005 | Supabase af-south-1 region availability on free tier — confirm or identify fallback region | Tech Lead | Open |
| OI-006 | Multi-tenancy model: schema-per-tenant vs. row-level-security-only — decision needed before DB migration design | Architect | Open |
| OI-007 | WhatsApp chatbot AI: use Upstash Vector + Claude API vs. Dialogflow — evaluate cost at free tier scale | Tech Lead | Open |
| OI-008 | Payroll: integrate with KRA iTax for PAYE filing or export-only for Phase 1? | Finance Lead | Open |

---

*Document prepared using the Capability Canvas framework (Domain → SubDomain → Capability → Epic → Feature → User Story) as defined in the attached `.cypher` graph model.*

*Next step: Review with pilot school principal and bursar → prioritise EPIC 1 feature backlog → begin Go API scaffold and Supabase schema migrations.*

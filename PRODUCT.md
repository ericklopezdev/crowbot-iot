# Crowbot — Product Platform

> Companion to [`PLAN.md`](./PLAN.md). `PLAN.md` is the **distributed-backend
> roadmap** (edge → MQTT → AI pipeline → persistence → k3s). This document is the
> **product layer** built on top of it: the parent-facing platform that turns a
> child's robot conversations into a developmental dashboard.

## Vision

Crowbot is a learning robot for families. A parent buys the robot, claims it with
a serial / QR code, and assigns it to a child. The child learns by **talking and
asking questions** — an interactive alternative to a screen — while the robot
quietly helps the parent **understand and support the child's cognitive
development** through a dashboard: usage over time, a graph of the topics the
child is curious about, growth by cognitive area, and concrete recommendations.

The mission framing: help the next generation develop cognitively instead of
being passive on a phone, and give parents insight + guidance to support it.

## Product flow

```
Factory ──> Device pre-provisioned (serial_number + activation_code/QR), unclaimed
Parent  ──> signs up (account)
        ──> claims device (scan QR / type serial)  → device bound to account
        ──> creates child profile(s)
        ──> assigns device to a child              → active assignment
Child   ──> talks to the robot                     → interactions (existing pipeline)
Server  ──> device → account → ACTIVE child         → attributes each interaction
        ──> async analysis classifies each turn     → topics + cognitive area
Parent  ──> dashboard: usage · interest graph · areas · recommendations
```

### Multi-child decision

**One active child per device at a time.** An account can have several children;
a device is assigned to exactly one child via an `active` row in
`device_assignments` (history preserved). The parent re-assigns in the app. This
gives 100%-reliable attribution with no unreliable on-device child identification
and no firmware change. (Rejected alternatives: robot asks "who are you?" per
session — unreliable identification + firmware work; one robot permanently per
child — forces one purchase per child.)

### Attribution & trust

The device id travels in the MQTT topic (`/device/{id}/...`, per `PLAN.md`
Stage 2). The **server** resolves `device → account → active child` from
`device_assignments` — it does **not** trust a `kid_id` sent by the device. The
existing `audio/start` `kid_id` field is ignored for attribution (kept only as an
optional hint / deprecated).

---

## Architecture (added on top of PLAN.md)

```
                       ┌──────────────────────────────────────┐
   ESP32 ──MQTT──────► │ mqtt ingest + orchestrator (existing) │
                       └───────────────┬──────────────────────┘
                                       │ persists interaction (+ child_id)
                                       ▼
                          ┌────────────────────────┐
                          │ store (pgx + sqlc)     │
                          └───┬───────────────┬────┘
              enqueue analysis│               │
                              ▼               ▼
                  ┌───────────────────┐   ┌──────────────────┐  REST  ┌───────────┐
                  │ analysis worker   │   │ api (cmd/api)    │ ─────► │ dashboard │
                  │ LLM classifier →  │   │ auth · devices · │        │  (React)  │
                  │ topics + area +   │   │ children ·       │        └───────────┘
                  │ stats + recos     │   │ dashboard reads  │
                  └───────────────────┘   └──────────────────┘
```

The analysis worker reuses the existing vendor-neutral `LLMService` interface —
it is the **data/ML axis** of the flagship: structured classification of every
conversation turn into developmental signal.

---

## Data model (PostgreSQL — `migrations/0002_product.sql`)

Builds on the `devices` / `interactions` tables from `PLAN.md` Stage 3.

```sql
-- Parents / accounts
accounts(
  id uuid pk, email citext unique, password_hash text,
  full_name text, created_at, updated_at)

-- Robots (extends the Stage 3 devices table)
devices(
  id uuid pk,                     -- = MQTT device id
  serial_number text unique,      -- printed on the box
  activation_code_hash text,      -- what the QR encodes; stored hashed
  account_id uuid fk null,        -- null = unclaimed
  name text,                      -- "Living room robot"
  mqtt_username text, mqtt_password_hash text,
  firmware_version text, last_seen_at, claimed_at, created_at)

-- Children
children(
  id uuid pk, account_id uuid fk,
  name text, birthdate date,      -- age → tunes LLM level & safety
  avatar text, created_at)

-- Which child uses which robot (history kept)
device_assignments(
  id uuid pk, device_id fk, child_id fk,
  active bool, assigned_at, unassigned_at)
  -- partial unique index: one active assignment per device

-- Interactions (extends Stage 3 with child_id + session)
interactions(
  id, device_id fk, child_id fk, session_id uuid,
  transcript text, response_text text,
  stt_latency_ms, llm_latency_ms, tts_latency_ms, total_latency_ms,
  llm_model, llm_tokens_in, llm_tokens_out, created_at)

-- Analysis layer: turns conversations into "development"
interaction_analysis(
  interaction_id pk fk,
  cognitive_area text,      -- language | logic-math | science |
                            --   socio-emotional | creativity | social-world
  question_type text,       -- curiosity | homework | emotional | play
  complexity_level smallint,
  sentiment text,
  model text, analyzed_at)

-- Topic taxonomy (the graph "nodes")
topics(id, slug unique, label, parent_id fk null, area text)
interaction_topics(interaction_id fk, topic_id fk)   -- N:N bridge

-- Materialized for fast interest-graph reads
child_topic_stats(
  child_id fk, topic_id fk,
  interaction_count int, first_seen_at, last_seen_at,
  pk(child_id, topic_id))

-- Recommendations generated for the parent
recommendations(
  id, child_id fk, topic_id fk null, area text null,
  title text, body text, status text,   -- new | read | dismissed
  generated_at)
```

Usage charts (days × hours heatmap) are computed on the fly from `interactions`
(`date_trunc('hour', created_at)` grouped by `child_id`); a materialized
`daily_usage` table can be added later if it gets heavy.

---

## Backend (Go) — added components

Built on the existing `internal/store` (sqlc) and `internal/api` packages.

### a) Auth & accounts
- Hand-rolled JWT (access + refresh), `bcrypt` password hashing.
- Middleware injects `account_id` into `context.Context`; all account-scoped
  endpoints enforce ownership (a parent only ever sees their own children/devices).

### b) REST API (`internal/api`)
```
POST /api/auth/signup · login · refresh
POST /api/devices/claim        {serial_number, activation_code}
GET  /api/devices
POST /api/devices/{id}/assign  {child_id}
CRUD /api/children
-- Dashboard reads:
GET /api/children/{id}/overview         → KPIs + usage heatmap
GET /api/children/{id}/usage?from&to    → time series
GET /api/children/{id}/topics           → nodes + edges (interest graph)
GET /api/children/{id}/areas            → development by cognitive area
GET /api/children/{id}/recommendations
GET /api/children/{id}/interactions     → paginated conversation feed
```

### c) Analysis worker (data/ML axis)
- After an `interaction` is persisted, enqueue an analysis job (worker pool over a
  channel, or an **outbox** table so jobs survive restarts).
- The worker sends the `transcript` to the LLM with a **structured-classification
  prompt** returning JSON `{cognitive_area, topics[], question_type, complexity,
  sentiment}`.
- Writes `interaction_analysis` + `interaction_topics`, upserts
  `child_topic_stats`, and (by threshold/cron) generates `recommendations`.
- Reuses the vendor-neutral `LLMService` — provider stays swappable.

### Per-child prompting (product-critical)
The global `SystemPrompt` in `internal/config` becomes **per-child**:
parameterized by age (from `birthdate`) and wrapped in **child-safe content
guardrails**. This is core product value, not a detail.

---

## Dashboard (React)

Stack: **Vite + React + TS · TanStack Query · Tailwind · Recharts · React Flow**.

Pages:
- **Auth** — login / signup.
- **Onboarding** — claim robot (type serial or scan QR), create child, assign.
- **Home** — child selector.
- **Child dashboard** (tabs):
  - *Summary* — usage heatmap (days × hours) + KPIs (total time, # questions,
    day streak).
  - *Interests* — topic-node graph (node size = frequency, color = area); click →
    detail.
  - *Areas* — radar / bars by cognitive area + parent recommendations.
  - *Topic detail* — that topic's conversations + "what they're learning / how to
    support it".

---

## Privacy (minors)

This is children's data. Design must include explicit parent opt-in to store
transcripts and keep COPPA / GDPR-K in mind (data minimization, retention limits,
deletion). Flagged here; not blocking the initial build.

---

## Phases (each = one PR, leaves `main` green)

| Phase | Scope | Builds on |
|-------|-------|-----------|
| **P1** | `accounts`, `devices` (claim), `children`, `device_assignments`, JWT auth, `child_id` attribution on interactions | PLAN.md Stage 3 |
| **P2** | Analysis worker + `interaction_analysis` + `topics` + `child_topic_stats` | new (data/ML) |
| **P3** | Dashboard read endpoints (overview / usage / topics / areas) | API |
| **P4** | React dashboard (auth + onboarding + tabs) | frontend |
| **P5** | `recommendations` generation + per-age prompt + safety guardrails | refinement |

**Sequencing:** finish the in-flight `refactor/foundations` work and PLAN.md
Stage 3 persistence first, then land P1 directly on top — don't fork mid-stage.

## Out of scope (for now)
- On-device child identification (deferred — single active child per device).
- Billing / subscriptions.
- Multi-parent (co-guardian) accounts.
- Real-time push of insights (polling first; websockets later).

# Crowbot — Engineering Roadmap

> Working roadmap to evolve Crowbot from an IoT prototype into a portfolio-grade
> **distributed backend system**. Each stage is a self-contained PR that leaves
> `main` green and the system runnable.

## Vision

A distributed, edge-to-cloud voice backend in Go: ESP32 devices stream audio over
MQTT to a concurrent, session-based pipeline (STT → LLM → TTS) behind a
**vendor-neutral AI layer**, with every interaction persisted to PostgreSQL,
exposed over a REST API, observable via Prometheus, containerized, and deployed
on Kubernetes (k3s on-prem, EKS-portable).

## Architecture (target)

```
┌─────────┐  MQTT   ┌────────────────────────────┐
│ ESP32 / │ ──────► │ mqtt ingest (cmd/mqtt)     │
│ N devs  │ ◄────── │  session router (per-device,│
└─────────┘ chunks  │  concurrency-safe, context) │
                    └─────────────┬───────────────┘
                                  │ orchestrator: STT→LLM→TTS
                                  │ (per-stage latency timing)
                    ┌─────────────┼──────────────────┐
                    ▼             ▼                   ▼
            ┌──────────────┐ ┌─────────┐    ┌──────────────────┐
            │ AI layer     │ │ store   │    │ api (cmd/api)    │ REST
            │ (interfaces) │ │ pgx +   │◄───│  history/metrics │ ──► dashboard
            │ gcp|mock|loc │ │ sqlc    │    └──────────────────┘
            └──────────────┘ └────┬────┘
                                  │
                                  ▼
                       ┌────────────────────┐
                       │ Prometheus + slog  │
                       └────────────────────┘
```

## Tech decisions

| Concern            | Choice                          | Rationale |
|--------------------|---------------------------------|-----------|
| DB driver          | **pgx/v5**                      | Idiomatic, fast, native Postgres types |
| SQL → Go structs   | **sqlc**                        | Write plain SQL, get type-safe structs. No ORM magic. |
| Migrations         | **goose**, `migrations/*.sql`   | Versioned, plain SQL, separate from queries |
| MQTT audio format  | **binary payload + device-id in topic** | No base64 (+33% on a constrained MCU); enables device-side streaming playback |
| AI layer           | GCP managed behind interfaces   | Quality kept; provider is a swappable detail, not the cloud skill |
| Local AI testing   | `mock` + optional `local` impls | Deterministic CI + zero-cost local runs |
| Logging            | `log/slog` (structured)         | JSON logs, observability-ready |
| Deploy             | k3s (Kubernetes), EKS-portable  | Most transferable ops skill, $0 on-prem |

> **Alternative considered:** plain `database/sql` + manual `rows.Scan`. Rejected
> in favor of sqlc to keep handwritten SQL while eliminating boilerplate and
> getting compile-time-checked structs.

## Target layout

```
.
├── cmd/{api,mqtt,local,devicesim}/  # entrypoints (devicesim = laptop "fake ESP32")
├── internal/
│   ├── core/                    # orchestrator (+ per-stage timing)
│   ├── mqtt/                    # session router + handlers
│   ├── services/                # STT/TTS/LLM interfaces + impls:
│   │                            #   gcp_*, mock_*, local_*
│   ├── store/                   # sqlc-generated code + db access
│   ├── api/                     # HTTP handlers (real)
│   └── config/                  # env config loading
├── db/queries/                  # sqlc query files (.sql)
├── migrations/                  # 0001_schema.sql, ...
├── testdata/audio/              # 6 sample .wav files for MQTT tests
├── deploy/{docker,k8s}/         # Dockerfiles + k3s manifests
├── sqlc.yaml
├── docker-compose.yml
└── .github/workflows/ci.yml
```

---

## Stage 1 — Foundations (fix) 🔧

**Goal:** make the existing code correct and reviewable. This is the first thing
a senior backend reviewer inspects; nothing else lands until it's done.

**Status: mostly done** — only the English README remains.

**Tasks**
- [x] Replace the global mutable `currentDevice` with a **per-device session
  router** keyed by `deviceID` (`internal/mqtt/client.go`). Handles N concurrent
  devices safely (covered by `-race` test).
- [x] **Fix MQTT transport reliability (server side):** QoS 1 subscriptions +
  binary sequenced chunk protocol (device id from topic, `[u16 index][u16 total]
  [PCM]`). Server reassembles by `index`, detects missing chunks, logs loss.
- [x] Remove the hardcoded credential filename; load config via `internal/config`
  from env (`CROWBOT_*_PROVIDER`, `MQTT_BROKER`, creds path, API key).
- [x] Propagate `context.Context` through the orchestrator and all service calls.
- [x] Add unit tests with mock STT/LLM/TTS (`mock_*.go`) — orchestrator + session
  router + chunk framing + topic parsing.
- [x] Delete dead code (`extractDeviceID` → replaced by `deviceIDFromTopic`).
- [x] **Isolate PortAudio behind `//go:build portaudio`** so server/tests/CI build
  without the system lib; only `cmd/local` (real mic) needs `-tags portaudio`.
- [ ] Rewrite `README.md` in **English**.

**Acceptance**
- [x] `go build ./...`, `go vet ./...` clean (golangci-lint deferred to Stage 5).
- [x] `go test -race ./...` passes; orchestrator + session router covered.
- [x] Concurrent sessions for N devices with no data races.
- [x] Out-of-order chunks reassemble correctly; a dropped chunk is detected and
  errored (not silently lost).

### Firmware backlog (deferred — requires hardware) 📟

Tracked here, **not implemented until the ESP32 is back**. Validated against
mocks + the audio harness in the meantime; hardware testing when available.

White noise (capture):
- **A1** — I2S comm format: switch `I2S_COMM_FORMAT_I2S_MSB` →
  `I2S_COMM_FORMAT_STAND_I2S` (INMP441 is standard I2S; MSB-only misaligns 1 bit).
- **A2** — Verify INMP441 L/R pin wiring matches `I2S_CHANNEL_FMT_ONLY_LEFT`
  (L/R must be tied to GND for left channel).
- **A3** — Discard the first 2–3 DMA reads after `i2s_start` (mic settle time).
- **A4** — `i2s_read` timeout (50 ms) < chunk fill time (64 ms for 1024 samples);
  use `portMAX_DELAY` or drop `CHUNK_SAMPLES` to 512 (~32 ms).

Missing chunks (firmware side):
- **B2-tx** — Emit the binary sequenced chunk protocol (Stage 2 wire format),
  removing the base64 encode + the 3 KB `base64Buffer`.
- **B3** — Smaller chunks (512 samples) + respect AsyncMqttClient flow control to
  avoid TCP out-buffer overflow / silent publish drops.

> Playback / storage design (streaming, no SD card) lives in **Stage 2**.

---

## Stage 2 — Device protocol & streaming playback (edge) 🔊

**Goal:** lock the device-facing wire contract and the playback model **before**
building persistence + the API on top of it. The server side is implementable now
(no hardware); the firmware playback is hardware-gated (see Stage 1 backlog) but
designed here so the contract is final and stable.

### Wire format (both directions)

Device id travels in the **topic**, not the payload → the server uses wildcard
subscriptions (`/device/+/audio/chunk`) and stays stateless about a "current device".

Upstream (device → server):
- `/device/{id}/audio/start` → JSON `{"kid_id":"..."}`   (once)
- `/device/{id}/audio/chunk` → binary `[u16 index LE][u16 total LE][PCM 16-bit]`
- `/device/{id}/audio/end`   → empty                      (once)

Downstream (server → device):
- `/device/{id}/audio/response_chunk` → binary `[u16 index LE][u16 total LE][PCM]`
- `/device/{id}/audio/response_end`   → empty

Rationale: raw PCM avoids base64's +33% size and the ~3 KB encode/decode buffers on
a ~200 KB-RAM MCU; per chunk it's a `memcpy`, not `sprintf` + base64.

### Streaming playback — why no SD card is needed

Today the firmware **accumulates the entire response in RAM** (`realloc` per chunk)
and only then plays it. At 16 kHz/16-bit = **32 KB/s** with ~200 KB usable heap,
responses cap at **~6 s** before `realloc` fails — that pressure is what (wrongly)
pushes toward an SD card.

Fix: **stream each response chunk as it arrives** through an I2S DMA ring buffer and
free it after playback. RAM stays **constant (~8–16 KB)** regardless of response
length. The binary `index/total` protocol makes this clean (play in order, with a
tiny reorder buffer for the rare out-of-order chunk).

**Status: server side done.** The binary protocol is implemented both directions
in `internal/mqtt/client.go`, and `cmd/devicesim` is a Go reference implementation
(laptop "fake ESP32") that speaks it end-to-end. Firmware playback stays deferred.

**Tasks**
- [x] Server: consume the binary upstream format and emit the binary
  downstream/response format; wildcard subscriptions; device id from topic.
- [x] `cmd/devicesim`: Go reference client (sends WAV as binary chunks, reassembles
  the binary response) — doubles as the firmware contract reference.
- [ ] *(Firmware — deferred, needs hardware)* replace accumulate-then-play with
  streaming playback via I2S DMA + small ring buffer; drop the internal 8-bit DAC.

**Hardware decision (deferred):**
- Recommended: add a **MAX98357A** I2S amplifier (~$3) → native DMA streaming **and**
  real 16-bit output (vs the current internal 8-bit DAC quality loss).
- Alternative: an **ESP32-WROVER** with 4–8 MB PSRAM (~$5) to keep the simpler
  buffer-everything model (8 MB ≈ 250 s). Less elegant; no streaming.
- Rejected: **SD card** — extra SPI hardware, latency, and complexity for a problem
  a 16 KB ring buffer solves.

**Acceptance**
- [x] Server publishes/parses the binary protocol end-to-end; `devicesim` and
  `test_mqtt.sh` round-trip it.
- [ ] *(When hardware available)* device plays an arbitrarily long response with
  constant RAM and no SD card.

---

## Stage 3 — Persistence + API 🗄️

**Goal:** persist every interaction and expose it over REST. This creates the
dataset that powers the data/ML axis later.

**Tasks**
- Add Postgres via **pgx/v5**; connection pool in `internal/store`.
- Write `migrations/0001_schema.sql` (applied with **goose**). Initial schema:
  - `devices(id, name, created_at)`
  - `interactions(id, device_id, kid_id, transcript, response_text,
    stt_latency_ms, llm_latency_ms, tts_latency_ms, total_latency_ms,
    llm_model, llm_tokens_in, llm_tokens_out, created_at)`
- Write SQL queries in `db/queries/*.sql`; generate type-safe structs with
  **sqlc** (`sqlc.yaml`).
- Instrument the orchestrator to time each stage (STT/LLM/TTS) and persist one
  `interactions` row per turn.
- Build the real **`cmd/api`** HTTP server (`internal/api`):
  - `GET /healthz`
  - `GET /api/devices`
  - `GET /api/devices/{id}/interactions`
  - `GET /api/interactions/{id}`
  - `GET /api/metrics/summary` (counts, avg latencies)

**Acceptance**
- A run through the MQTT pipeline writes a complete `interactions` row.
- API returns interaction history as JSON.
- sqlc codegen is reproducible (`sqlc generate` produces no diff in CI).

---

## Stage 4 — Local testing: mock/local AI + 6-audio MQTT harness 🎧

**Goal:** run and validate the full pipeline on the laptop with **zero GCP cost**
and deterministic results — the day-to-day testing path while the ESP32 hardware
is unavailable. A minimal mock + harness is bootstrapped in Stage 1 (to test the
transport fixes); this stage hardens it into the 6-audio suite and provider switch.

> Mock STT returns canned transcripts (validates transport / sequencing /
> persistence / API deterministically), **not** real transcription quality. Swap
> `CROWBOT_*_PROVIDER=gcp` when you want real STT/TTS or hardware testing.

**Tasks**
- Add `mock` implementations of the STT/TTS interfaces:
  - `MockSTT`: maps each known test audio → a canned transcript.
  - `MockTTS`: returns a pre-recorded/placeholder WAV.
- Add provider selection via env: `CROWBOT_STT_PROVIDER=gcp|mock|local`
  (same for TTS/LLM). Showcases the vendor-neutral abstraction.
- Add **6 sample audio files** under `testdata/audio/`.
- Provide an MQTT test harness (extend `test_mqtt.sh` or a Go `cmd`/test) that
  publishes the 6 audios as chunked sessions and verifies responses come back.
- *(Optional stretch)* `local` real implementations via a Python sidecar
  (e.g. **vosk** for STT, **piper** for TTS) exposed over a small HTTP service,
  called from Go. Only if quality/effort is worth it — otherwise mocks suffice.

**Acceptance**
- `CROWBOT_*_PROVIDER=mock go run cmd/mqtt/main.go` + harness runs end-to-end
  with no cloud calls.
- All 6 audios produce persisted `interactions` rows (when run against Stage 3).

---

## Stage 5 — CI (GitHub Actions) 🤖

**Goal:** automated quality gate on every push/PR.

**Tasks**
- `.github/workflows/ci.yml`: `go vet`, `golangci-lint`, `sqlc generate --check`,
  `go test -race ./...` with a Postgres service container, coverage report.
- Add a multi-stage `Dockerfile` per service and a `docker-compose.yml`
  (mosquitto + postgres + services) for one-command local bring-up.
- Add graceful shutdown (`signal.NotifyContext`) to all entrypoints.

**Acceptance**
- CI is green on PRs; coverage badge in README.
- `docker compose up` brings up the whole stack locally.

---

## Stage 6 — Observability (local first) 📊

**Goal:** make the system measurable before deploying it.

**Tasks**
- Expose Prometheus metrics: per-stage latency histograms, request/error
  counters, active-session gauge, throughput.
- Structured logging with `slog` (JSON), request/session correlation IDs.
- Local Prometheus + Grafana via docker-compose; a starter dashboard.
- *(Data/ML)* basic LLM eval/usage analytics queries over `interactions`
  (latency percentiles, token usage, response stats).

**Acceptance**
- `/metrics` endpoint scraped by local Prometheus.
- Grafana dashboard shows live pipeline latencies.

---

## Stage 7 — Deploy (k3s) 🚀

**Goal:** run the stack on Kubernetes; primary CV deployment story.

**Tasks**
- k8s manifests in `deploy/k8s/`: Deployments, Services, Ingress, PVC (Postgres),
  Secrets/ConfigMaps, health/readiness probes.
- Deploy to **k3s** on the laptop.
- Document EKS portability (what changes for AWS EKS); optional one-off EKS deploy
  to back the claim.

**Acceptance**
- `kubectl apply -k deploy/k8s` brings up the stack on k3s.
- Pipeline works end-to-end against the cluster; dashboard reachable via Ingress.

---

## Out of scope (future)

- Web dashboard UI polish (basic functional version in Stage 3/6).
- Self-hosted LLM (Ollama) on-prem.
- Multi-tenant auth / RBAC.
- Real ESP32 firmware integration tests.

## Execution rules

- One stage = one PR off `main`, small focused commits.
- `main` always builds and tests green.
- Decisions and weaknesses tracked in commit messages and this file.

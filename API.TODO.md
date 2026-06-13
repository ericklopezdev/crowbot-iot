# API.TODO — Backend (Go, net/http + pgx)

Phase **P1** of PRODUCT.md: parent accounts, device claiming, children, assignment.
Plain `net/http` (Go 1.22 routing), `pgxpool`, hand-rolled JWT auth. Check off as we go.

## Foundations
- [x] `internal/store/pool.go` — pgxpool constructor + ping
- [x] `internal/config` — `LoadAPI()` (DATABASE_URL, JWT_SECRET, API_ADDR)
- [x] `internal/auth/password.go` — bcrypt hash/verify
- [x] `internal/auth/jwt.go` — issue/parse access + refresh tokens

## HTTP plumbing
- [x] `internal/api/response.go` — JSON encode/decode/error helpers
- [x] `internal/api/middleware.go` — recover, request logging (slog), requireAuth
- [x] `internal/api/server.go` — Server struct + route table

## Endpoints
- [x] `GET  /healthz`
- [x] `POST /api/auth/signup`
- [x] `POST /api/auth/login`
- [x] `POST /api/auth/refresh`
- [x] `POST /api/devices/claim`        (serial + activation code → bind + MQTT creds)
- [x] `GET  /api/devices`
- [x] `POST /api/devices/{id}/assign`  (tx: deactivate old + assign child)
- [x] `POST /api/children`
- [x] `GET  /api/children`
- [x] `GET  /api/children/{id}`

## Wiring & verification
- [x] `cmd/api/main.go` — config → pool → tokens → server + graceful shutdown
- [x] `go build ./...` + `go vet ./...` clean
- [x] Smoke test against the compose DB (signup → claim → child → assign)

## Deferred (later phases)
- [ ] Analysis worker + dashboard read endpoints (PRODUCT.md P2/P3)
- [ ] MQTT ingest persists interactions with resolved `child_id`
- [ ] Unit/integration tests + CI

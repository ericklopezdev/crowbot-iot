-- +goose Up
-- Core voice-pipeline schema: devices and the interactions they produce.
-- This is the technical backend layer (PLAN.md Stage 3); the product layer
-- (accounts, children, analysis) is added in 0002.

-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
-- +goose StatementEnd

CREATE TABLE devices (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE interactions (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id        uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    transcript       text NOT NULL,
    response_text    text NOT NULL,
    stt_latency_ms   integer NOT NULL DEFAULT 0,
    llm_latency_ms   integer NOT NULL DEFAULT 0,
    tts_latency_ms   integer NOT NULL DEFAULT 0,
    total_latency_ms integer NOT NULL DEFAULT 0,
    llm_model        text NOT NULL DEFAULT '',
    llm_tokens_in    integer NOT NULL DEFAULT 0,
    llm_tokens_out   integer NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_interactions_device ON interactions (device_id, created_at DESC);

-- +goose Down
DROP TABLE interactions;
DROP TABLE devices;

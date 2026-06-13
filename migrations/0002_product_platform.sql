-- +goose Up
-- Product platform: parent accounts, device claiming, children, the active
-- device->child assignment, the LLM analysis layer, and parent recommendations.
-- See PRODUCT.md for the full data model.

-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;  -- case-insensitive email
-- +goose StatementEnd

-- Parents / accounts -------------------------------------------------------
CREATE TABLE accounts (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext NOT NULL UNIQUE,
    password_hash text NOT NULL,
    full_name     text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Extend devices for the claim/provisioning flow ---------------------------
ALTER TABLE devices
    ADD COLUMN serial_number        text UNIQUE,
    ADD COLUMN activation_code_hash text,
    ADD COLUMN account_id           uuid REFERENCES accounts(id) ON DELETE SET NULL,
    ADD COLUMN mqtt_username        text,
    ADD COLUMN mqtt_password_hash   text,
    ADD COLUMN firmware_version     text NOT NULL DEFAULT '',
    ADD COLUMN last_seen_at         timestamptz,
    ADD COLUMN claimed_at           timestamptz;

CREATE INDEX idx_devices_account ON devices (account_id);

-- Children -----------------------------------------------------------------
CREATE TABLE children (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name       text NOT NULL,
    birthdate  date,
    avatar     text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_children_account ON children (account_id);

-- Which child currently uses which robot (history preserved) ---------------
CREATE TABLE device_assignments (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id     uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    child_id      uuid NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    active        boolean NOT NULL DEFAULT true,
    assigned_at   timestamptz NOT NULL DEFAULT now(),
    unassigned_at timestamptz
);

-- Enforce: at most one ACTIVE assignment per device.
CREATE UNIQUE INDEX uq_device_active_assignment
    ON device_assignments (device_id) WHERE active;
CREATE INDEX idx_device_assignments_child ON device_assignments (child_id);

-- Attribute interactions to a child + a session ----------------------------
ALTER TABLE interactions
    ADD COLUMN child_id   uuid REFERENCES children(id) ON DELETE SET NULL,
    ADD COLUMN session_id uuid;

CREATE INDEX idx_interactions_child ON interactions (child_id, created_at DESC);

-- Analysis layer: structured developmental signal per interaction ----------
CREATE TABLE interaction_analysis (
    interaction_id   uuid PRIMARY KEY REFERENCES interactions(id) ON DELETE CASCADE,
    cognitive_area   text NOT NULL DEFAULT '',  -- language|logic-math|science|socio-emotional|creativity|social-world
    question_type    text NOT NULL DEFAULT '',  -- curiosity|homework|emotional|play
    complexity_level smallint NOT NULL DEFAULT 0,
    sentiment        text NOT NULL DEFAULT '',
    model            text NOT NULL DEFAULT '',
    analyzed_at      timestamptz NOT NULL DEFAULT now()
);

-- Topic taxonomy (graph nodes) ---------------------------------------------
CREATE TABLE topics (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug      text NOT NULL UNIQUE,
    label     text NOT NULL,
    parent_id uuid REFERENCES topics(id) ON DELETE SET NULL,
    area      text NOT NULL DEFAULT ''
);

CREATE TABLE interaction_topics (
    interaction_id uuid NOT NULL REFERENCES interactions(id) ON DELETE CASCADE,
    topic_id       uuid NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (interaction_id, topic_id)
);

-- Materialized per-child topic frequencies (fast interest-graph reads) -----
CREATE TABLE child_topic_stats (
    child_id          uuid NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    topic_id          uuid NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    interaction_count integer NOT NULL DEFAULT 0,
    first_seen_at     timestamptz NOT NULL DEFAULT now(),
    last_seen_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (child_id, topic_id)
);

-- Recommendations surfaced to the parent -----------------------------------
CREATE TABLE recommendations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id     uuid NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    topic_id     uuid REFERENCES topics(id) ON DELETE SET NULL,
    area         text NOT NULL DEFAULT '',
    title        text NOT NULL,
    body         text NOT NULL,
    status       text NOT NULL DEFAULT 'new',  -- new|read|dismissed
    generated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_recommendations_child ON recommendations (child_id, status);

-- +goose Down
DROP TABLE recommendations;
DROP TABLE child_topic_stats;
DROP TABLE interaction_topics;
DROP TABLE topics;
DROP TABLE interaction_analysis;

ALTER TABLE interactions DROP COLUMN child_id, DROP COLUMN session_id;

DROP TABLE device_assignments;
DROP TABLE children;

ALTER TABLE devices
    DROP COLUMN serial_number,
    DROP COLUMN activation_code_hash,
    DROP COLUMN account_id,
    DROP COLUMN mqtt_username,
    DROP COLUMN mqtt_password_hash,
    DROP COLUMN firmware_version,
    DROP COLUMN last_seen_at,
    DROP COLUMN claimed_at;

DROP TABLE accounts;

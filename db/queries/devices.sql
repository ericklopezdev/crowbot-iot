-- name: SeedDevice :one
-- Factory provisioning: create an unclaimed device with a serial + activation hash.
INSERT INTO devices (serial_number, activation_code_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetDeviceBySerial :one
SELECT * FROM devices WHERE serial_number = $1;

-- name: GetDeviceByID :one
SELECT * FROM devices WHERE id = $1;

-- name: ClaimDevice :one
-- Bind an unclaimed device to an account and provision its MQTT credentials.
UPDATE devices
SET account_id = $2,
    mqtt_username = $3,
    mqtt_password_hash = $4,
    claimed_at = now()
WHERE id = $1 AND account_id IS NULL
RETURNING *;

-- name: ListDevicesByAccount :many
SELECT * FROM devices WHERE account_id = $1 ORDER BY claimed_at DESC;

-- name: TouchDeviceLastSeen :exec
UPDATE devices SET last_seen_at = now() WHERE id = $1;

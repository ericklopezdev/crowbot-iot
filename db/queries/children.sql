-- name: CreateChild :one
INSERT INTO children (account_id, name, birthdate, avatar)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetChild :one
SELECT * FROM children WHERE id = $1;

-- name: ListChildrenByAccount :many
SELECT * FROM children WHERE account_id = $1 ORDER BY created_at;

-- name: AssignDevice :one
-- Deactivate any current assignment for this device, then create the new one.
-- (Run inside a tx with DeactivateDeviceAssignments first.)
INSERT INTO device_assignments (device_id, child_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeactivateDeviceAssignments :exec
UPDATE device_assignments
SET active = false, unassigned_at = now()
WHERE device_id = $1 AND active;

-- name: GetActiveChildForDevice :one
-- Resolve device -> active child (used by the MQTT ingest for attribution).
SELECT c.*
FROM device_assignments da
JOIN children c ON c.id = da.child_id
WHERE da.device_id = $1 AND da.active;

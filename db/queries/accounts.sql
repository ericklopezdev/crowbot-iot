-- name: CreateAccount :one
INSERT INTO accounts (email, password_hash, full_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAccountByEmail :one
SELECT * FROM accounts WHERE email = $1;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

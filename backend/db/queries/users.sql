-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
-- name: CreateUser :one
INSERT INTO users (email, name, image) VALUES ($1, $2, $3) RETURNING *;
-- name: UpdateUser :one
UPDATE users SET name = $2, image = $3 WHERE id = $1 RETURNING *;
-- name: UpsertUser :one
INSERT INTO users (email, name, image) VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET name = $2, image = $3 RETURNING *;

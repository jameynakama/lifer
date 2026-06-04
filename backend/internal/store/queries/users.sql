-- Columns are enumerated (no SELECT */RETURNING *) so that adding a column to
-- users is an explicit decision per query, not a silent change to API output.

-- name: GetUserByGoogleID :one
SELECT id, google_id, email, name, picture, is_admin, created_at
FROM users WHERE google_id = $1;

-- name: GetUserByID :one
SELECT id, google_id, email, name, picture, is_admin, created_at
FROM users WHERE id = $1;

-- GetUsers feeds the admin users API: google_id is deliberately excluded so
-- third-party identifiers never reach the client.
-- name: GetUsers :many
SELECT id, email, name, picture, is_admin, created_at FROM users;

-- name: SetUserIsAdmin :exec
UPDATE users SET is_admin = $2 WHERE id = $1;

-- name: UpsertUser :one
INSERT INTO users (google_id, email, name, picture)
VALUES ($1, $2, $3, $4)
ON CONFLICT (google_id) DO UPDATE
    SET email   = EXCLUDED.email,
        name    = EXCLUDED.name,
        picture = EXCLUDED.picture
RETURNING id, google_id, email, name, picture, is_admin, created_at;

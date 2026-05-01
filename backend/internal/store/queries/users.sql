-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpsertUser :one
INSERT INTO users (google_id, email, name, picture)
VALUES ($1, $2, $3, $4)
ON CONFLICT (google_id) DO UPDATE
    SET email   = EXCLUDED.email,
        name    = EXCLUDED.name,
        picture = EXCLUDED.picture
RETURNING *;

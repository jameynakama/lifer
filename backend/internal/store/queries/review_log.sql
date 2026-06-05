-- name: CreateReviewLog :one
INSERT INTO review_log (user_id, species_code, lane, rating, guessed_species_code, media_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

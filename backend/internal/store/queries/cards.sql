-- name: GetDueCards :many
SELECT c.*, r.file_path AS recording_path, s.common_name, s.scientific_name
FROM cards c
JOIN recordings r ON r.id = c.recording_id
JOIN species s ON s.id = r.species_id
WHERE c.user_id = $1 AND c.due <= NOW()
ORDER BY c.due
LIMIT $2;

-- name: UpsertCard :one
INSERT INTO cards (user_id, recording_id)
VALUES ($1, $2)
ON CONFLICT (user_id, recording_id) DO NOTHING
RETURNING *;

-- name: UpdateCardSchedule :one
UPDATE cards
SET stability   = $3,
    difficulty  = $4,
    due         = $5,
    last_review = NOW(),
    reps        = reps + 1,
    lapses      = $6,
    state       = $7
WHERE user_id = $1 AND recording_id = $2
RETURNING *;

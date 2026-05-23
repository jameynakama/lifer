-- name: GetNextDueCard :one
SELECT c.id, c.user_id, c.species_id, c.lane,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.state, c.created_at,
       s.common_name, s.scientific_name
FROM cards c
JOIN species s ON s.id = c.species_id
JOIN group_species gs ON gs.species_id = c.species_id
WHERE c.user_id = $1
  AND gs.group_id = $2
  AND c.lane = $3
  AND c.due <= NOW()
ORDER BY c.due
LIMIT 1;

-- name: GetRandomRecording :one
SELECT file_path FROM species_recordings
WHERE species_id = $1
ORDER BY random()
LIMIT 1;

-- name: GetRandomImage :one
SELECT file_path FROM species_images
WHERE species_id = $1
ORDER BY random()
LIMIT 1;

-- name: GetCard :one
SELECT id, user_id, species_id, lane, stability, difficulty, due,
       last_review, reps, lapses, state, created_at
FROM cards
WHERE user_id = $1 AND species_id = $2 AND lane = $3;

-- name: UpdateCardSchedule :one
UPDATE cards
SET stability   = $4,
    difficulty  = $5,
    due         = $6,
    last_review = NOW(),
    reps        = reps + 1,
    lapses      = $7,
    state       = $8
WHERE user_id = $1 AND species_id = $2 AND lane = $3
RETURNING id, user_id, species_id, lane, stability, difficulty, due,
          last_review, reps, lapses, state, created_at;

-- name: UpsertCard :exec
INSERT INTO cards (user_id, species_id, lane)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, species_id, lane) DO NOTHING;

-- name: DeleteCard :exec
DELETE FROM cards
WHERE user_id = $1 AND species_id = $2 AND lane = $3;

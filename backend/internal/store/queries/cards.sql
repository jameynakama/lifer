-- name: GetNextDueCard :one
SELECT c.id, c.user_id, c.species_code, c.lane,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.state, c.created_at,
       s.common_name, s.scientific_name
FROM cards c
JOIN species s ON s.ebird_code = c.species_code
JOIN group_species gs ON gs.species_code = c.species_code
WHERE c.user_id = $1
  AND gs.group_id = $2
  AND c.lane = $3
  AND c.due <= NOW()
ORDER BY c.due
LIMIT 1;

-- name: GetRandomRecording :one
SELECT file_path, type FROM species_recordings
WHERE species_code = $1 AND quality IN ('A', 'B')
ORDER BY random()
LIMIT 1;

-- name: CountDueCards :one
SELECT COUNT(*)
FROM cards c
JOIN group_species gs ON gs.species_code = c.species_code
WHERE c.user_id = $1
  AND gs.group_id = $2
  AND c.lane = $3
  AND c.due <= NOW();

-- name: GetRandomImage :one
SELECT file_path FROM species_images
WHERE species_code = $1
ORDER BY random()
LIMIT 1;

-- name: GetCard :one
SELECT id, user_id, species_code, lane, stability, difficulty, due,
       last_review, reps, lapses, state, created_at
FROM cards
WHERE user_id = $1 AND species_code = $2 AND lane = $3;

-- name: UpdateCardSchedule :one
UPDATE cards
SET stability   = $4,
    difficulty  = $5,
    due         = $6,
    last_review = NOW(),
    reps        = reps + 1,
    lapses      = $7,
    state       = $8
WHERE user_id = $1 AND species_code = $2 AND lane = $3
RETURNING id, user_id, species_code, lane, stability, difficulty, due,
          last_review, reps, lapses, state, created_at;

-- name: UpsertCard :exec
INSERT INTO cards (user_id, species_code, lane)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, species_code, lane) DO NOTHING;

-- name: DeleteCard :exec
DELETE FROM cards
WHERE user_id = $1 AND species_code = $2 AND lane = $3;

-- name: GetGroupPracticeCards :many
SELECT s.ebird_code, s.common_name, s.scientific_name,
       COALESCE(
           (SELECT file_path FROM species_recordings
            WHERE species_code = s.ebird_code AND quality IN ('A', 'B')
            ORDER BY random() LIMIT 1),
           ''
       )::text AS audio_url,
       COALESCE(
           (SELECT file_path FROM species_images
            WHERE species_code = s.ebird_code
            ORDER BY random() LIMIT 1),
           ''
       )::text AS image_url
FROM species s
JOIN group_species gs ON gs.species_code = s.ebird_code
WHERE gs.group_id = $1
ORDER BY s.common_name;

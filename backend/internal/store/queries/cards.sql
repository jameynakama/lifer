-- name: GetNextDueCard :one
SELECT c.id, c.user_id, c.species_code, c.lane,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.state, c.created_at,
       s.common_name, s.scientific_name
FROM cards c
JOIN species s ON s.ebird_code = c.species_code
JOIN deck_species ds ON ds.species_code = c.species_code
LEFT JOIN user_species_preferences usp
       ON usp.user_id = c.user_id AND usp.species_code = c.species_code
WHERE c.user_id = $1
  AND ds.deck_id = $2
  AND c.lane = $3
  AND c.due <= NOW()
  AND (
    ($3 = 'audio' AND COALESCE(usp.audio_enabled, true))
    OR
    ($3 = 'image' AND COALESCE(usp.image_enabled, true))
  )
ORDER BY c.due
LIMIT 1;

-- name: GetRandomRecording :one
SELECT file_path, type, credit FROM species_recordings
WHERE species_code = $1 AND quality IN ('A', 'B')
ORDER BY random()
LIMIT 1;

-- name: CountDueCards :one
SELECT COUNT(*)
FROM cards c
JOIN deck_species ds ON ds.species_code = c.species_code
LEFT JOIN user_species_preferences usp
       ON usp.user_id = c.user_id AND usp.species_code = c.species_code
WHERE c.user_id = $1
  AND ds.deck_id = $2
  AND c.lane = $3
  AND c.due <= NOW()
  AND (
    ($3 = 'audio' AND COALESCE(usp.audio_enabled, true))
    OR
    ($3 = 'image' AND COALESCE(usp.image_enabled, true))
  );

-- name: GetRandomImage :one
SELECT file_path, credit FROM species_images
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

-- name: UpsertCardsForDeck :exec
INSERT INTO cards (user_id, species_code, lane)
SELECT $1, ds.species_code, l.lane
FROM deck_species ds
CROSS JOIN (VALUES ('audio'::text), ('image'::text)) AS l(lane)
WHERE ds.deck_id = $2
ON CONFLICT (user_id, species_code, lane) DO NOTHING;

-- name: GetDeckPracticeCards :many
SELECT s.ebird_code, s.common_name, s.scientific_name,
       COALESCE(rec.file_path, '') AS audio_url,
       COALESCE(rec.credit, '')    AS audio_credit,
       COALESCE(img.file_path, '') AS image_url,
       COALESCE(img.credit, '')    AS image_credit
FROM species s
JOIN deck_species ds ON ds.species_code = s.ebird_code
LEFT JOIN LATERAL (
    SELECT file_path, credit FROM species_recordings
    WHERE species_code = s.ebird_code AND quality IN ('A', 'B')
    ORDER BY random() LIMIT 1
) rec ON true
LEFT JOIN LATERAL (
    SELECT file_path, credit FROM species_images
    WHERE species_code = s.ebird_code
    ORDER BY random() LIMIT 1
) img ON true
WHERE ds.deck_id = $1
ORDER BY s.common_name;

-- name: BulkUpsertCards :exec
INSERT INTO cards (user_id, species_code, lane)
SELECT $1, s.code, l.lane
FROM unnest($2::text[]) AS s(code)
CROSS JOIN (VALUES ('audio'::text), ('image'::text)) AS l(lane)
ON CONFLICT (user_id, species_code, lane) DO NOTHING;

-- name: GetNextDueAt :one
SELECT due AS next_due_at
FROM cards
WHERE user_id = $1
  AND due > NOW()
ORDER BY due
LIMIT 1;

-- name: GetNextDueCard :one
SELECT c.id, c.user_id, c.species_code, c.lane,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.state, c.created_at,
       s.common_name, s.scientific_name,
       COUNT(*) OVER () AS due_remaining
FROM cards c
JOIN species s ON s.ebird_code = c.species_code
JOIN deck_species ds ON ds.species_code = c.species_code
LEFT JOIN user_species_preferences usp
       ON usp.user_id = c.user_id AND usp.species_code = c.species_code
WHERE c.user_id = $1
  AND ds.deck_id = $2
  AND c.lane = $3
  -- due_before pins a quiz session to its start time: cards FSRS re-dues
  -- mid-session (1-10min learning steps) don't repeat within the session.
  AND c.due <= COALESCE(sqlc.narg('due_before'), NOW())
  AND (
    ($3 = 'audio' AND COALESCE(usp.audio_enabled, true))
    OR
    ($3 = 'image' AND COALESCE(usp.image_enabled, true))
  )
  -- Never serve a card the quiz can't render: the species must have media for
  -- the lane (recordings must meet the A/B quality bar GetRandomRecording uses).
  AND (
    ($3 = 'audio' AND EXISTS (
      SELECT 1 FROM species_recordings sr
      WHERE sr.species_code = c.species_code AND sr.quality IN ('A', 'B')))
    OR
    ($3 = 'image' AND EXISTS (
      SELECT 1 FROM species_images si
      WHERE si.species_code = c.species_code))
  )
-- Bucket due-time by the minute, then shuffle within the bucket. A fresh deck
-- seeds every card with an identical `due`; a plain `ORDER BY c.due` left the
-- tiebreak to the scan order, so the quiz replayed the same species sequence
-- every session. Minute granularity keeps FSRS's 1-10min learning steps in
-- order while randomising the (always-tied) fresh cards.
ORDER BY date_trunc('minute', c.due), random()
LIMIT 1;

-- GetRandomMediaForSpecies picks a random quiz-quality recording and a random
-- image in one round trip (same LATERAL pattern as GetDeckPracticeCards).
-- Missing media comes back as empty strings.
-- name: GetRandomMediaForSpecies :one
SELECT COALESCE(rec.file_path, '')      AS audio_path,
       COALESCE(rec.type, '')           AS audio_type,
       COALESCE(rec.credit, '')         AS audio_credit,
       COALESCE(rec.xeno_canto_id, '')  AS audio_id,
       COALESCE(img.file_path, '')      AS image_path,
       COALESCE(img.credit, '')         AS image_credit,
       COALESCE(img.macaulay_id, '')    AS image_id
FROM (SELECT $1::text AS code) sp
LEFT JOIN LATERAL (
    SELECT file_path, type, credit, xeno_canto_id FROM species_recordings
    WHERE species_code = sp.code AND quality IN ('A', 'B')
    ORDER BY random() LIMIT 1
) rec ON true
LEFT JOIN LATERAL (
    SELECT file_path, credit, macaulay_id FROM species_images
    WHERE species_code = sp.code
    ORDER BY random() LIMIT 1
) img ON true;

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
       COALESCE(rec.file_path, '')     AS audio_url,
       COALESCE(rec.credit, '')        AS audio_credit,
       COALESCE(rec.xeno_canto_id, '') AS audio_id,
       COALESCE(img.file_path, '')     AS image_url,
       COALESCE(img.credit, '')        AS image_credit,
       COALESCE(img.macaulay_id, '')   AS image_id
FROM species s
JOIN deck_species ds ON ds.species_code = s.ebird_code
LEFT JOIN LATERAL (
    SELECT file_path, credit, xeno_canto_id FROM species_recordings
    WHERE species_code = s.ebird_code AND quality IN ('A', 'B')
    ORDER BY random() LIMIT 1
) rec ON true
LEFT JOIN LATERAL (
    SELECT file_path, credit, macaulay_id FROM species_images
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

-- Stats: per-card bucket counts. Buckets per the stats spec: not_seen = never
-- reviewed; known = FSRS Review state; relearning = lapsed; else learning.
-- Preference-disabled lanes are excluded to match GetNextDueCard: a lane the
-- user toggled off is never served, so its cards stay reps = 0 forever and
-- would otherwise haunt the progress bar as permanent, unclearable not_seen.
-- name: GetCardStateCounts :many
SELECT
    CASE
        -- reps = 0 wins by definition: a never-reviewed card is not_seen regardless of state.
        WHEN c.reps = 0  THEN 'not_seen'
        WHEN c.state = 2 THEN 'known'
        WHEN c.state = 3 THEN 'relearning'
        ELSE 'learning'
    END AS bucket,
    COUNT(*) AS count
FROM cards c
LEFT JOIN user_species_preferences usp
       ON usp.user_id = c.user_id AND usp.species_code = c.species_code
WHERE c.user_id = $1
  AND c.lane = COALESCE(sqlc.narg('lane'), c.lane)
  AND (
    (c.lane = 'audio' AND COALESCE(usp.audio_enabled, true))
    OR
    (c.lane = 'image' AND COALESCE(usp.image_enabled, true))
  )
  -- Only count species still in one of the user's decks. Removing a species
  -- from a deck leaves its card behind; that orphan is unservable (GetNextDueCard
  -- joins deck_species) and would otherwise linger as a permanent not_seen.
  AND EXISTS (
    SELECT 1 FROM deck_species ds
    JOIN decks d ON d.id = ds.deck_id
    WHERE ds.species_code = c.species_code AND d.owner_id = c.user_id
  )
GROUP BY bucket;

-- Totals mirror GetCardStateCounts' lane-preference filter so disabled lanes
-- don't inflate card/species/review counts.
-- name: GetCardTotals :one
SELECT COUNT(DISTINCT c.species_code)      AS species,
       COUNT(*)                          AS cards,
       COALESCE(SUM(c.reps), 0)::bigint    AS reviews,
       COALESCE(SUM(c.lapses), 0)::bigint  AS lapses
FROM cards c
LEFT JOIN user_species_preferences usp
       ON usp.user_id = c.user_id AND usp.species_code = c.species_code
WHERE c.user_id = $1
  AND c.lane = COALESCE(sqlc.narg('lane'), c.lane)
  AND (
    (c.lane = 'audio' AND COALESCE(usp.audio_enabled, true))
    OR
    (c.lane = 'image' AND COALESCE(usp.image_enabled, true))
  )
  AND EXISTS (
    SELECT 1 FROM deck_species ds
    JOIN decks d ON d.id = ds.deck_id
    WHERE ds.species_code = c.species_code AND d.owner_id = c.user_id
  );

-- Stats: known cards with FSRS fields for retrievability math in Go.
-- "Known" here (state = 2) is intentionally equivalent to GetCardStateCounts'
-- known bucket: FSRS cannot produce state 2 with reps = 0, so the two
-- predicates cannot diverge. Keep them in sync if either changes -- including
-- the lane-preference filter below, so a known-then-disabled card doesn't show
-- in Fading/Remember while vanishing from the progress bar.
-- name: GetKnownCards :many
SELECT c.species_code, s.common_name, s.scientific_name, c.lane,
       c.stability, c.due, c.last_review
FROM cards c
JOIN species s ON s.ebird_code = c.species_code
LEFT JOIN user_species_preferences usp
       ON usp.user_id = c.user_id AND usp.species_code = c.species_code
WHERE c.user_id = $1
  AND c.state = 2
  AND c.lane = COALESCE(sqlc.narg('lane'), c.lane)
  AND (
    (c.lane = 'audio' AND COALESCE(usp.audio_enabled, true))
    OR
    (c.lane = 'image' AND COALESCE(usp.image_enabled, true))
  )
  AND EXISTS (
    SELECT 1 FROM deck_species ds
    JOIN decks d ON d.id = ds.deck_id
    WHERE ds.species_code = c.species_code AND d.owner_id = c.user_id
  );

-- Stats: species known in exactly one lane, biggest stability gap first.
-- Both lanes must be enabled for a gap to be actionable -- if the weak lane is
-- disabled, the user opted out of practicing it, so it's not a gap to surface.
-- name: GetLaneGaps :many
SELECT a.species_code, s.common_name, s.scientific_name,
       CASE WHEN a.state = 2 THEN 'audio' ELSE 'image' END AS known_lane,
       CASE WHEN a.state = 2 THEN 'image' ELSE 'audio' END AS weak_lane,
       ABS(a.stability - i.stability)::float AS stability_gap
FROM cards a
JOIN cards i   ON i.user_id = a.user_id AND i.species_code = a.species_code AND i.lane = 'image'
JOIN species s ON s.ebird_code = a.species_code
LEFT JOIN user_species_preferences usp
       ON usp.user_id = a.user_id AND usp.species_code = a.species_code
WHERE a.user_id = $1
  AND a.lane = 'audio'
  AND COALESCE(usp.audio_enabled, true)
  AND COALESCE(usp.image_enabled, true)
  AND EXISTS (
    SELECT 1 FROM deck_species ds
    JOIN decks d ON d.id = ds.deck_id
    WHERE ds.species_code = a.species_code AND d.owner_id = a.user_id
  )
  AND ((a.state = 2 AND i.state <> 2) OR (i.state = 2 AND a.state <> 2))
ORDER BY stability_gap DESC
LIMIT 10;

-- name: DeleteAllCardsForUser :execrows
DELETE FROM cards
WHERE user_id = $1;

-- Re-seed blank cards for every species in the user's decks (both lanes,
-- minus preference-disabled ones). Used by reset: nothing else re-creates
-- cards for already-added species, so without this a reset leaves existing
-- decks permanently card-less.
-- name: SeedCardsForUserDecks :execrows
INSERT INTO cards (user_id, species_code, lane)
SELECT DISTINCT d.owner_id, ds.species_code, l.lane
FROM decks d
JOIN deck_species ds ON ds.deck_id = d.id
CROSS JOIN (VALUES ('audio'::text), ('image'::text)) AS l(lane)
LEFT JOIN user_species_preferences p
       ON p.user_id = d.owner_id AND p.species_code = ds.species_code
WHERE d.owner_id = sqlc.arg(user_id)::bigint
  AND ((l.lane = 'audio' AND COALESCE(p.audio_enabled, TRUE))
    OR (l.lane = 'image' AND COALESCE(p.image_enabled, TRUE)))
ON CONFLICT (user_id, species_code, lane) DO NOTHING;

-- Seeder: every card of the user's due at the given instant, all decks,
-- both lanes (the quiz path's GetNextDueCard is deck- and lane-scoped).
-- Mirrors the quiz's media filter: never simulate a review the real app
-- could not serve.
-- name: GetDueCardsForUser :many
SELECT id, user_id, species_code, lane, stability, difficulty, due,
       last_review, reps, lapses, state, created_at
FROM cards c
WHERE c.user_id = $1 AND c.due <= sqlc.arg(as_of)
  AND (
    (c.lane = 'audio' AND EXISTS (
      SELECT 1 FROM species_recordings sr
      WHERE sr.species_code = c.species_code AND sr.quality IN ('A', 'B')))
    OR
    (c.lane = 'image' AND EXISTS (
      SELECT 1 FROM species_images si
      WHERE si.species_code = c.species_code))
  )
ORDER BY c.due, c.id;

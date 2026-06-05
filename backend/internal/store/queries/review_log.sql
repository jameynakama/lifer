-- name: CreateReviewLog :one
INSERT INTO review_log (user_id, species_code, lane, rating, guessed_species_code, media_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- Stats: actual misidentifications only (skips have NULL guesses).
-- name: GetConfusionPairs :many
SELECT rl.species_code,
       a.common_name     AS actual_common_name,
       a.scientific_name AS actual_scientific_name,
       rl.guessed_species_code::text AS guessed_species_code,
       g.common_name     AS guessed_common_name,
       g.scientific_name AS guessed_scientific_name,
       COUNT(*)          AS count
FROM review_log rl
JOIN species a ON a.ebird_code = rl.species_code
JOIN species g ON g.ebird_code = rl.guessed_species_code
WHERE rl.user_id = $1
  AND rl.guessed_species_code IS NOT NULL
  AND rl.guessed_species_code <> rl.species_code
  AND rl.lane = COALESCE(sqlc.narg('lane'), rl.lane)
GROUP BY rl.species_code, a.common_name, a.scientific_name,
         rl.guessed_species_code, g.common_name, g.scientific_name
ORDER BY count DESC, rl.species_code
LIMIT 10;

-- Stats: accuracy by eBird family; species without a backfilled family are omitted.
-- name: GetFamilyAccuracy :many
SELECT s.family::text AS family,
       COUNT(*) AS attempts,
       COUNT(*) FILTER (WHERE rl.rating = 3) AS correct
FROM review_log rl
JOIN species s ON s.ebird_code = rl.species_code
WHERE rl.user_id = $1
  AND s.family IS NOT NULL
  AND rl.lane = COALESCE(sqlc.narg('lane'), rl.lane)
GROUP BY s.family
ORDER BY (COUNT(*) FILTER (WHERE rl.rating = 3))::float / COUNT(*) ASC, attempts DESC
LIMIT 10;

-- Stats: specific media the user keeps missing (>=3 looks). media_url resolves
-- opportunistically -- deleted media yields ''.
-- name: GetHardMedia :many
SELECT rl.media_id::text AS media_id,
       rl.lane,
       rl.species_code,
       s.common_name,
       s.scientific_name,
       COALESCE(MAX(sr.file_path), MAX(si.file_path), '')::text AS media_url,
       COUNT(*) AS attempts,
       COUNT(*) FILTER (WHERE rl.rating = 3) AS correct
FROM review_log rl
JOIN species s ON s.ebird_code = rl.species_code
LEFT JOIN species_recordings sr ON rl.lane = 'audio' AND sr.xeno_canto_id = rl.media_id
LEFT JOIN species_images si    ON rl.lane = 'image' AND si.macaulay_id  = rl.media_id
WHERE rl.user_id = $1
  AND rl.media_id IS NOT NULL
  AND rl.lane = COALESCE(sqlc.narg('lane'), rl.lane)
GROUP BY rl.media_id, rl.lane, rl.species_code, s.common_name, s.scientific_name
HAVING COUNT(*) >= 3
ORDER BY (COUNT(*) FILTER (WHERE rl.rating = 3))::float / COUNT(*) ASC, attempts DESC
LIMIT 10;

-- name: GetReviewAccuracy :one
SELECT COUNT(*) AS attempts,
       COUNT(*) FILTER (WHERE rating = 3) AS correct
FROM review_log
WHERE user_id = $1
  AND lane = COALESCE(sqlc.narg('lane'), lane);

-- name: CountReviewsSince :one
SELECT COUNT(*)
FROM review_log
WHERE user_id = $1
  AND reviewed_at > $2
  AND lane = COALESCE(sqlc.narg('lane'), lane);

-- name: DeleteAllReviewsForUser :execrows
DELETE FROM review_log
WHERE user_id = $1;

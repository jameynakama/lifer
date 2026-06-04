-- name: SearchSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
WHERE common_name ILIKE '%' || $1 || '%'
   OR scientific_name ILIKE '%' || $1 || '%'
   OR ebird_code ILIKE '%' || $1 || '%'
ORDER BY common_name
LIMIT 50;

-- name: ListSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    COUNT(*) OVER() AS total_count,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
ORDER BY common_name
LIMIT $1 OFFSET $2;

-- name: GetSpeciesByCode :one
SELECT ebird_code, common_name, scientific_name
FROM species
WHERE ebird_code = $1;

-- name: GetSpeciesRecordings :many
SELECT xeno_canto_id, file_path, quality, type, credit, locked
FROM species_recordings
WHERE species_code = $1
ORDER BY quality, type;

-- name: GetSpeciesImages :many
SELECT macaulay_id, file_path, credit, locked
FROM species_images
WHERE species_code = $1;

-- name: GetDecksForSpecies :many
SELECT deck_id
FROM deck_species
WHERE species_code = $1
  AND deck_id IN (SELECT id FROM decks WHERE owner_id = $2);

-- name: ListAllSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
ORDER BY common_name;

-- name: GetImageByID :one
SELECT macaulay_id, species_code, file_path, credit, locked, created_at
FROM species_images
WHERE macaulay_id = $1;

-- name: GetRecordingByID :one
SELECT xeno_canto_id, species_code, file_path, quality, type, credit, locked, created_at
FROM species_recordings
WHERE xeno_canto_id = $1;

-- name: DeleteImage :exec
DELETE FROM species_images WHERE macaulay_id = $1;

-- name: DeleteRecording :exec
DELETE FROM species_recordings WHERE xeno_canto_id = $1;

-- name: SetRecordingLocked :exec
UPDATE species_recordings SET locked = $2 WHERE xeno_canto_id = $1;

-- name: SetImageLocked :exec
UPDATE species_images SET locked = $2 WHERE macaulay_id = $1;

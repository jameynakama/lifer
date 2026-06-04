-- name: UpsertSpecies :one
INSERT INTO species (ebird_code, common_name, scientific_name)
VALUES ($1, $2, $3)
ON CONFLICT (ebird_code) DO UPDATE
    SET common_name     = EXCLUDED.common_name,
        scientific_name = EXCLUDED.scientific_name
RETURNING *;

-- "locked" protects media from REMOVAL only (see the cleanup path and the
-- NOT locked delete guards below); ingest may still add new media to a locked
-- bird and refresh same-source fields on existing rows.
-- name: UpsertRecording :one
INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (xeno_canto_id) DO UPDATE
    SET file_path = EXCLUDED.file_path,
        quality   = EXCLUDED.quality,
        type      = EXCLUDED.type,
        credit    = EXCLUDED.credit
RETURNING *;

-- name: UpsertSpeciesImage :one
INSERT INTO species_images (macaulay_id, species_code, file_path, credit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (macaulay_id) DO UPDATE
    SET file_path = EXCLUDED.file_path,
        credit    = EXCLUDED.credit
RETURNING *;

-- name: ListIncompleteSpecies :many
SELECT ebird_code FROM species
WHERE ebird_code NOT IN (SELECT DISTINCT species_code FROM species_recordings)
   OR ebird_code NOT IN (SELECT DISTINCT species_code FROM species_images);

-- name: ListCompleteSpeciesEbirdCodes :many
SELECT s.ebird_code
FROM species s
WHERE EXISTS (SELECT 1 FROM species_recordings r WHERE r.species_code = s.ebird_code)
  AND EXISTS (SELECT 1 FROM species_images si WHERE si.species_code = s.ebird_code);

-- name: DeleteRecordingsBySpeciesCode :exec
DELETE FROM species_recordings WHERE species_code = $1 AND NOT locked;

-- name: DeleteSpeciesImagesBySpeciesCode :exec
DELETE FROM species_images WHERE species_code = $1 AND NOT locked;

-- name: DeleteSpeciesByCode :exec
DELETE FROM species WHERE ebird_code = $1;

-- Species with any locked media are protected from ingest cleanup: deleting
-- the species row would cascade onto locked rows, so cleanup must skip them.
-- name: ListSpeciesCodesWithLockedMedia :many
SELECT DISTINCT species_code FROM species_recordings WHERE locked
UNION
SELECT DISTINCT species_code FROM species_images WHERE locked;

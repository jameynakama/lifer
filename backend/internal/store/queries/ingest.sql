-- name: UpsertSpecies :one
INSERT INTO species (common_name, scientific_name, ebird_code)
VALUES ($1, $2, $3)
ON CONFLICT (ebird_code) DO UPDATE
    SET common_name     = EXCLUDED.common_name,
        scientific_name = EXCLUDED.scientific_name
RETURNING *;

-- name: UpsertRecording :one
INSERT INTO species_recordings (species_id, xeno_canto_id, file_path, quality, type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (xeno_canto_id) DO UPDATE
    SET file_path = EXCLUDED.file_path,
        quality   = EXCLUDED.quality,
        type      = EXCLUDED.type
RETURNING *;

-- name: UpsertSpeciesImage :one
INSERT INTO species_images (species_id, macaulay_id, file_path, credit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (macaulay_id) DO UPDATE
    SET file_path = EXCLUDED.file_path,
        credit    = EXCLUDED.credit
RETURNING *;

-- name: ListIncompleteSpecies :many
SELECT id, ebird_code FROM species
WHERE id NOT IN (SELECT DISTINCT species_id FROM species_recordings)
   OR id NOT IN (SELECT DISTINCT species_id FROM species_images);

-- name: ListCompleteSpeciesEbirdCodes :many
SELECT s.ebird_code
FROM species s
WHERE EXISTS (SELECT 1 FROM species_recordings r WHERE r.species_id = s.id)
  AND EXISTS (SELECT 1 FROM species_images si WHERE si.species_id = s.id);

-- name: DeleteRecordingsBySpeciesID :exec
DELETE FROM species_recordings WHERE species_id = $1;

-- name: DeleteSpeciesImagesBySpeciesID :exec
DELETE FROM species_images WHERE species_id = $1;

-- name: DeleteSpeciesByID :exec
DELETE FROM species WHERE id = $1;

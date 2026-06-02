-- name: SearchSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
WHERE common_name ILIKE '%' || $1 || '%'
   OR scientific_name ILIKE '%' || $1 || '%'
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
SELECT xeno_canto_id, file_path, quality, type
FROM species_recordings
WHERE species_code = $1
ORDER BY quality, type;

-- name: GetSpeciesImages :many
SELECT macaulay_id, file_path, credit
FROM species_images
WHERE species_code = $1;

-- name: GetGroupsForSpecies :many
SELECT group_id
FROM group_species
WHERE species_code = $1
  AND group_id IN (SELECT id FROM groups WHERE owner_id = $2);

-- name: ListAllSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
ORDER BY common_name;

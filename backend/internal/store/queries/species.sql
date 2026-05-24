-- name: SearchSpecies :many
SELECT id, common_name, scientific_name, ebird_code
FROM species
WHERE common_name ILIKE '%' || $1 || '%'
   OR scientific_name ILIKE '%' || $1 || '%'
ORDER BY common_name
LIMIT 20;

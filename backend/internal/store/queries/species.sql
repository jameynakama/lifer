-- name: SearchSpecies :many
SELECT ebird_code, common_name, scientific_name
FROM species
WHERE common_name ILIKE '%' || $1 || '%'
   OR scientific_name ILIKE '%' || $1 || '%'
ORDER BY common_name
LIMIT 20;

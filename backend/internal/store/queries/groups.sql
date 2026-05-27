-- name: ListUserGroups :many
SELECT g.id, g.name, g.description, g.is_preset, g.owner_id, g.created_at,
    COUNT(CASE WHEN c.lane = 'audio' AND c.due <= NOW() THEN 1 END) AS audio_due,
    COUNT(CASE WHEN c.lane = 'image' AND c.due <= NOW() THEN 1 END) AS image_due
FROM groups g
LEFT JOIN group_species gs ON gs.group_id = g.id
LEFT JOIN cards c ON c.species_code = gs.species_code AND c.user_id = $1
WHERE g.owner_id = $1
GROUP BY g.id
ORDER BY g.name;

-- name: GetGroup :one
SELECT id, name, description, is_preset, owner_id, created_at
FROM groups
WHERE id = $1;

-- name: CreateGroup :one
INSERT INTO groups (name, owner_id)
VALUES ($1, $2)
RETURNING id, name, description, is_preset, owner_id, created_at;

-- name: UpdateGroupName :one
UPDATE groups SET name = $2 WHERE id = $1
RETURNING id, name, description, is_preset, owner_id, created_at;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: ListGroupSpecies :many
SELECT s.ebird_code, s.common_name, s.scientific_name
FROM species s
JOIN group_species gs ON gs.species_code = s.ebird_code
WHERE gs.group_id = $1
ORDER BY s.common_name;

-- name: AddSpeciesToGroup :exec
INSERT INTO group_species (group_id, species_code)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveSpeciesFromGroup :exec
DELETE FROM group_species
WHERE group_id = $1 AND species_code = $2;

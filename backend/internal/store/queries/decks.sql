-- name: ListUserDecks :many
SELECT d.id, d.name, d.description, d.owner_id, d.created_at,
    COUNT(CASE WHEN c.lane = 'audio' AND c.due <= NOW() THEN 1 END) AS audio_due,
    COUNT(CASE WHEN c.lane = 'image' AND c.due <= NOW() THEN 1 END) AS image_due
FROM decks d
LEFT JOIN deck_species ds ON ds.deck_id = d.id
LEFT JOIN cards c ON c.species_code = ds.species_code AND c.user_id = $1
WHERE d.owner_id = $1
GROUP BY d.id
ORDER BY d.name;

-- name: GetDeck :one
SELECT id, name, description, owner_id, created_at
FROM decks
WHERE id = $1;

-- name: GetDeckWithDue :one
SELECT d.id, d.name, d.description, d.owner_id, d.created_at,
    COUNT(CASE WHEN c.lane = 'audio' AND c.due <= NOW() THEN 1 END) AS audio_due,
    COUNT(CASE WHEN c.lane = 'image' AND c.due <= NOW() THEN 1 END) AS image_due
FROM decks d
LEFT JOIN deck_species ds ON ds.deck_id = d.id
LEFT JOIN cards c ON c.species_code = ds.species_code AND c.user_id = $2
WHERE d.id = $1
GROUP BY d.id;

-- name: CreateDeck :one
INSERT INTO decks (name, owner_id)
VALUES ($1, $2)
RETURNING id, name, description, owner_id, created_at;

-- name: UpdateDeckName :one
UPDATE decks SET name = $2 WHERE id = $1
RETURNING id, name, description, owner_id, created_at;

-- name: DeleteDeck :exec
DELETE FROM decks WHERE id = $1;

-- name: ListDeckSpeciesWithPrefs :many
SELECT s.ebird_code, s.common_name, s.scientific_name,
       COALESCE(p.audio_enabled, true) AS audio_enabled,
       COALESCE(p.image_enabled, true) AS image_enabled
FROM species s
JOIN deck_species ds ON ds.species_code = s.ebird_code
LEFT JOIN user_species_preferences p
       ON p.species_code = s.ebird_code AND p.user_id = $2
WHERE ds.deck_id = $1
ORDER BY s.common_name;

-- name: AddSpeciesToDeck :exec
INSERT INTO deck_species (deck_id, species_code)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveSpeciesFromDeck :exec
DELETE FROM deck_species
WHERE deck_id = $1 AND species_code = $2;

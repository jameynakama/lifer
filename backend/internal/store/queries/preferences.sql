-- name: UpsertPreferences :one
INSERT INTO user_species_preferences (user_id, species_code, audio_enabled, image_enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, species_code) DO UPDATE
SET audio_enabled = EXCLUDED.audio_enabled,
    image_enabled = EXCLUDED.image_enabled
RETURNING user_id, species_code, audio_enabled, image_enabled, created_at;

-- name: GetPreferences :one
SELECT user_id, species_code, audio_enabled, image_enabled, created_at
FROM user_species_preferences
WHERE user_id = $1 AND species_code = $2;

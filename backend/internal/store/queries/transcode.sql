-- Queries for cmd/transcode. Deliberately narrow: the job reads recordings and
-- writes peaks, nothing else. It must never reach ingest's cleanup queries,
-- which delete species rows DB-wide and cascade to cards and review_log.

-- name: ListRecordingsForTranscode :many
SELECT xeno_canto_id, species_code, file_path, peaks
FROM species_recordings
ORDER BY species_code, xeno_canto_id;

-- name: SetRecordingPeaks :exec
UPDATE species_recordings SET peaks = $2 WHERE xeno_canto_id = $1;

-- Precomputed waveform envelope, PeakCount buckets of 0..255 (see
-- internal/audio). NULL means not yet backfilled; the player falls back to
-- generated bars for those rows.
ALTER TABLE species_recordings ADD COLUMN peaks smallint[];

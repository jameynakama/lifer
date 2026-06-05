ALTER TABLE species ADD COLUMN family TEXT;

CREATE TABLE review_log (
    id                   BIGSERIAL   PRIMARY KEY,
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_code         TEXT        NOT NULL REFERENCES species(ebird_code) ON DELETE CASCADE,
    lane                 TEXT        NOT NULL CHECK (lane IN ('audio', 'image')),
    -- 1 = Again, 3 = Good (quiz auto-rates; correct ⇔ 3)
    rating               SMALLINT    NOT NULL CHECK (rating BETWEEN 1 AND 4),
    -- NULL = "I don't know" (the skip button). SET NULL so catalog cleanup
    -- can't turn real misidentifications into fake skips.
    guessed_species_code TEXT        REFERENCES species(ebird_code) ON DELETE SET NULL,
    -- xeno_canto_id or macaulay_id that was shown. No FK on purpose: it would
    -- need to reference two tables (discriminated by lane) and media is
    -- legitimately deleted/replaced. Joined opportunistically.
    media_id             TEXT,
    reviewed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_review_log_user_time ON review_log (user_id, reviewed_at);

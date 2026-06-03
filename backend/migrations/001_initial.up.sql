CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    google_id  TEXT        NOT NULL UNIQUE,
    email      TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    picture    TEXT        NOT NULL DEFAULT '',
    is_admin   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species (
    ebird_code      TEXT PRIMARY KEY,
    common_name     TEXT NOT NULL,
    scientific_name TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species_recordings (
    xeno_canto_id  TEXT    PRIMARY KEY,
    species_code   TEXT    NOT NULL REFERENCES species(ebird_code) ON DELETE CASCADE,
    file_path      TEXT    NOT NULL,
    quality        CHAR(1) NOT NULL CHECK (quality IN ('A','B','C','D','E')),
    type           TEXT    NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_species_recordings_species_code ON species_recordings(species_code);

CREATE TABLE species_images (
    macaulay_id  TEXT PRIMARY KEY,
    species_code TEXT NOT NULL REFERENCES species(ebird_code) ON DELETE CASCADE,
    file_path    TEXT NOT NULL,
    credit       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_species_images_species_code ON species_images(species_code);

CREATE TABLE decks (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    owner_id    BIGINT  REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE deck_species (
    deck_id      BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    species_code TEXT   NOT NULL REFERENCES species(ebird_code) ON DELETE CASCADE,
    PRIMARY KEY (deck_id, species_code)
);

CREATE TABLE cards (
    id           BIGSERIAL   PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_code TEXT        NOT NULL REFERENCES species(ebird_code) ON DELETE CASCADE,
    lane         TEXT        NOT NULL CHECK (lane IN ('audio', 'image')),
    stability    FLOAT       NOT NULL DEFAULT 0,
    difficulty   FLOAT       NOT NULL DEFAULT 0,
    due          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_review  TIMESTAMPTZ,
    reps         INT         NOT NULL DEFAULT 0,
    lapses       INT         NOT NULL DEFAULT 0,
    state        SMALLINT    NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, species_code, lane)
);
CREATE INDEX idx_cards_user_lane_due ON cards(user_id, lane, due);

CREATE TABLE user_species_preferences (
    user_id       BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_code  TEXT    NOT NULL REFERENCES species(ebird_code) ON DELETE CASCADE,
    audio_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    image_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, species_code)
);

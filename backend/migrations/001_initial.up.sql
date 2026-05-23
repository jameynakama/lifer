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
    id              BIGSERIAL PRIMARY KEY,
    common_name     TEXT NOT NULL,
    scientific_name TEXT NOT NULL,
    ebird_code      TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species_recordings (
    id             BIGSERIAL PRIMARY KEY,
    species_id     BIGINT      NOT NULL REFERENCES species(id),
    xeno_canto_id  TEXT        NOT NULL UNIQUE,
    file_path      TEXT        NOT NULL,
    quality        CHAR(1)     NOT NULL CHECK (quality IN ('A','B','C','D','E')),
    type           TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species_images (
    id           BIGSERIAL PRIMARY KEY,
    species_id   BIGINT      NOT NULL REFERENCES species(id),
    macaulay_id  TEXT        NOT NULL UNIQUE,
    file_path    TEXT        NOT NULL,
    credit       TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE groups (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    is_preset   BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id    BIGINT  REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE group_species (
    group_id   BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    species_id BIGINT NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, species_id)
);

CREATE TABLE cards (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id  BIGINT      NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    lane        TEXT        NOT NULL CHECK (lane IN ('audio', 'image')),
    stability   FLOAT       NOT NULL DEFAULT 0,
    difficulty  FLOAT       NOT NULL DEFAULT 0,
    due         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_review TIMESTAMPTZ,
    reps        INT         NOT NULL DEFAULT 0,
    lapses      INT         NOT NULL DEFAULT 0,
    state       SMALLINT    NOT NULL DEFAULT 0, -- 0=new 1=learning 2=review 3=relearning
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, species_id, lane)
);
CREATE INDEX idx_cards_user_lane_due ON cards(user_id, lane, due);
CREATE INDEX idx_species_recordings_species_id ON species_recordings(species_id);
CREATE INDEX idx_species_images_species_id ON species_images(species_id);

CREATE TABLE user_species_preferences (
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id    BIGINT      NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    audio_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    image_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, species_id)
);

ALTER TABLE species_recordings ADD COLUMN locked boolean NOT NULL DEFAULT false;
ALTER TABLE species_images ADD COLUMN locked boolean NOT NULL DEFAULT false;

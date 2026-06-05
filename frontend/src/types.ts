export interface Stat {
    label: string;
    value: string | number;
}

export interface BirdCard {
    ebird_code: string;
    common_name: string;
    scientific_name: string;
    media_url: string;
    photo_url: string;
    lane: 'audio' | 'image';
    recording_type: string;
    recording_credit: string;
    photo_credit: string;
    due_remaining: number;
}

export interface Deck {
    id: number;
    name: string;
    description?: string;
    audio_due: number;
    image_due: number;
}

export interface DecksResponse {
    decks: Deck[];
    next_due_at: string | null;
}

export interface PresetDeck {
    id: number;
    name: string;
    description: string;
    species_count: number;
}

export interface Species {
    ebird_code: string;
    common_name: string;
    scientific_name: string;
}

export interface SpeciesListItem {
    ebird_code: string;
    common_name: string;
    scientific_name: string;
    image_url: string | null;
}

/** A species as it appears in a user's deck, with per-user lane toggles. */
export interface DeckSpecies extends Species {
    audio_enabled: boolean;
    image_enabled: boolean;
}

/** One page of GET /api/v1/species (server-side pagination). */
export interface SpeciesPage {
    results: SpeciesListItem[];
    count: number;
    next: string | null;
    previous: string | null;
}

export interface SpeciesRecording {
    xeno_canto_id: string;
    file_path: string;
    quality: string;
    type: string;
    credit: string;
}

export interface SpeciesImage {
    macaulay_id: string;
    file_path: string;
    credit: string;
}

/** GET /api/v1/species/{ebird_code} */
export interface SpeciesDetail extends Species {
    recordings: SpeciesRecording[];
    images: SpeciesImage[];
}

/** Admin media rows additionally carry the removal-protection flag. */
export interface AdminSpeciesRecording extends SpeciesRecording {
    locked: boolean;
}

export interface AdminSpeciesImage extends SpeciesImage {
    locked: boolean;
}

/** A user-owned deck as the admin deck list sees it. */
export interface UserDeck {
    id: number;
    name: string;
    description: string;
    owner_name: string;
    owner_email: string;
    species_count: number;
}

/** GET /api/v1/admin/decks/{id}/species — deck header portion. */
export interface AdminDeckInfo {
    id: number;
    name: string;
    owner_name: string;
    owner_email: string;
}

/** The signed-in user as the auth store holds it (subset of /me). */
export interface SessionUser {
    id: number;
    email: string;
    name: string;
    is_admin: boolean;
}

/** GET /api/v1/admin/users row. */
export interface User {
  id: number;
  email: string;
  name: string;
  picture: string;
  is_admin: boolean;
  created_at: string;
}

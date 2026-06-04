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

export interface User {
  id: number;
  email: string;
  name: string;
  picture: string;
  is_admin: boolean;
  created_at: string;
}

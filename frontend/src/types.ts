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
    due_remaining: number;
}

export interface Deck {
    id: number;
    name: string;
    audio_due: number;
    image_due: number;
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

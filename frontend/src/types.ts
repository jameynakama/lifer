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
}

export interface Group {
    id: number;
    name: string;
    is_preset: boolean;
    audio_due: number;
    image_due: number;
}

export interface Species {
    ebird_code: string;
    common_name: string;
    scientific_name: string;
}

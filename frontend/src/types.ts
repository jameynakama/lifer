export interface Stat {
    label: string;
    value: string | number;
}

export interface BirdCard {
    id: string;
    recording_path: string;
    common_name: string;
    scientific_name: string;
    photo_path: string;
}

export interface Group {
    id: string;
    name: string;
    is_preset: boolean;
    due_count: number;
}

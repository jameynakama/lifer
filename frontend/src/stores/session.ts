import { writable, type Writable } from "svelte/store";

type Lane = 'audio' | 'image';

type Session = {
    groupId: string | null;
    lane: Lane | null;
};

export const session: Writable<Session> = writable({ groupId: null, lane: null });

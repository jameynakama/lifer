import { writable, type Writable } from "svelte/store";

type Session = { groupId: string | null };

export const session: Writable<Session> = writable({ groupId: null });

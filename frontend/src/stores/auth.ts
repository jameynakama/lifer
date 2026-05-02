import { writable, type Writable } from "svelte/store";

interface User {
    id: number | null;
    email: string | null;
    name: string | null;
}

type Auth = User | null;

export const auth: Writable<Auth> = writable(null);

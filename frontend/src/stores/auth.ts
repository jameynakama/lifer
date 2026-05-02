import { writable, type Writable } from "svelte/store";

interface User {
    id: number;
    email: string;
    name: string;
}

type Auth = User | null;

export const auth: Writable<Auth> = writable(null);

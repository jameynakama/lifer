import { writable, type Writable } from "svelte/store";

interface User {
    id: number;
    email: string;
    name: string;
    is_admin: boolean;
}

type Auth = User | null;

export const auth: Writable<Auth> = writable(null);

import { writable, type Writable } from "svelte/store";

type ViewStore = "login" | "dashboard" | "quiz";

export const view: Writable<ViewStore> = writable("login");

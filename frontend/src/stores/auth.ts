import { writable, type Writable } from 'svelte/store'
import type { SessionUser } from '../types'

type Auth = SessionUser | null

export const auth: Writable<Auth> = writable(null)

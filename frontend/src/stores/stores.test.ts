import { describe, it, expect } from 'vitest'
import { get } from 'svelte/store'
import { auth } from './auth'
import { view } from './view'
import { session } from './session'

describe('auth store', () => {
  it('starts as null', () => {
    expect(get(auth)).toBe(null)
  })
})

describe('view store', () => {
  it('starts as login', () => {
    expect(get(view)).toBe('login')
  })
})

describe('session store', () => {
  it('starts with null groupId', () => {
    expect(get(session).groupId).toBe(null)
  })
})

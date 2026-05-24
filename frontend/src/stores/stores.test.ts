import { describe, it, expect } from 'vitest'
import { get } from 'svelte/store'
import { auth } from './auth'

describe('auth store', () => {
  it('starts as null', () => {
    expect(get(auth)).toBe(null)
  })
})

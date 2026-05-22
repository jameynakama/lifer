import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { getCurrentTheme, initTheme, toggleTheme } from './theme'

const mockMatchMedia = (prefersDark: boolean) => {
  vi.stubGlobal('matchMedia', vi.fn().mockImplementation((query: string) => ({
    matches: query === '(prefers-color-scheme: dark)' ? prefersDark : !prefersDark,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })))
}

beforeEach(() => {
  document.documentElement.removeAttribute('data-theme')
  localStorage.clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('getCurrentTheme', () => {
  it('returns dark when data-theme is dark', () => {
    document.documentElement.setAttribute('data-theme', 'dark')
    mockMatchMedia(false)
    expect(getCurrentTheme()).toBe('dark')
  })

  it('returns light when data-theme is light', () => {
    document.documentElement.setAttribute('data-theme', 'light')
    mockMatchMedia(true)
    expect(getCurrentTheme()).toBe('light')
  })

  it('returns dark when OS prefers dark and no attribute', () => {
    mockMatchMedia(true)
    expect(getCurrentTheme()).toBe('dark')
  })

  it('returns light when OS prefers light and no attribute', () => {
    mockMatchMedia(false)
    expect(getCurrentTheme()).toBe('light')
  })
})

describe('initTheme', () => {
  it('sets data-theme from localStorage when saved', () => {
    localStorage.setItem('lifer-theme', 'light')
    mockMatchMedia(true)
    initTheme()
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('does not set data-theme when no localStorage preference', () => {
    mockMatchMedia(true)
    initTheme()
    expect(document.documentElement.getAttribute('data-theme')).toBeNull()
  })
})

describe('toggleTheme', () => {
  it('switches dark to light and persists', () => {
    document.documentElement.setAttribute('data-theme', 'dark')
    mockMatchMedia(true)
    toggleTheme()
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    expect(localStorage.getItem('lifer-theme')).toBe('light')
  })

  it('switches light to dark and persists', () => {
    document.documentElement.setAttribute('data-theme', 'light')
    mockMatchMedia(false)
    toggleTheme()
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('lifer-theme')).toBe('dark')
  })
})

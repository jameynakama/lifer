import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from '../stores/view'
import { session } from '../stores/session'
import Dashboard from './Dashboard.svelte'

describe('Dashboard', () => {
  beforeEach(() => {
    view.set('dashboard')
    session.set({ groupId: null, lane: null })
  })

  it('renders group names', () => {
    render(Dashboard)
    expect(screen.getAllByText(/pacific northwest/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
  })

  it('renders audio and image practice buttons', () => {
    render(Dashboard)
    expect(screen.getAllByRole('button', { name: /audio/i }).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: /image/i }).length).toBeGreaterThan(0)
  })

  it('sets session and switches to quiz when audio button clicked', async () => {
    render(Dashboard)
    await fireEvent.click(screen.getAllByRole('button', { name: /audio/i })[0])
    expect(get(session).groupId).toBeTruthy()
    expect(get(session).lane).toBe('audio')
    expect(get(view)).toBe('quiz')
  })

  it('sets session and switches to quiz when image button clicked', async () => {
    render(Dashboard)
    await fireEvent.click(screen.getAllByRole('button', { name: /image/i })[0])
    expect(get(session).groupId).toBeTruthy()
    expect(get(session).lane).toBe('image')
    expect(get(view)).toBe('quiz')
  })
})

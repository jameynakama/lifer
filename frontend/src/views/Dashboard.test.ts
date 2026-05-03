import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from '../stores/view'
import { session } from '../stores/session'
import Dashboard from './Dashboard.svelte'

describe('Dashboard', () => {
  beforeEach(() => {
    view.set('dashboard')
    session.set({ groupId: null })
  })

  it('renders a Start Practice button', () => {
    render(Dashboard)
    expect(screen.getByRole('button', { name: /start practice/i })).toBeInTheDocument()
  })

  it('renders the group with the most due cards prominently', () => {
    render(Dashboard)
    expect(screen.getAllByText(/pacific northwest/i).length).toBeGreaterThan(0)
  })

  it('sets session.groupId and switches to quiz when Start Practice clicked', async () => {
    render(Dashboard)
    await fireEvent.click(screen.getByRole('button', { name: /start practice/i }))
    expect(get(session).groupId).toBe('1')
    expect(get(view)).toBe('quiz')
  })

  it('switches to quiz when a group Practice button is clicked', async () => {
    render(Dashboard)
    const buttons = screen.getAllByRole('button', { name: /practice/i })
    await fireEvent.click(buttons[0])
    expect(get(view)).toBe('quiz')
  })
})

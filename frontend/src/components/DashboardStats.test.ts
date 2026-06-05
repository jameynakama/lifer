import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import DashboardStats from './DashboardStats.svelte'

afterEach(() => {
  vi.useRealTimers()
})

describe('DashboardStats', () => {
  it('renders all four tile labels', () => {
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt: null } })
    expect(screen.getByText('DUE TODAY')).toBeInTheDocument()
    expect(screen.getByText('AUDIO DUE')).toBeInTheDocument()
    expect(screen.getByText('IMAGE DUE')).toBeInTheDocument()
    expect(screen.getByText('NEXT DUE IN')).toBeInTheDocument()
  })

  it('shows totals in the first three tiles', () => {
    render(DashboardStats, { props: { audioDue: 3, imageDue: 5, nextDueAt: null } })
    expect(screen.getByText('8')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('shows "Now" with accent class when cards are due', () => {
    render(DashboardStats, {
      props: { audioDue: 2, imageDue: 1, nextDueAt: '2099-01-01T00:00:00Z' },
    })
    const el = screen.getByText('Now')
    expect(el).toBeInTheDocument()
    expect(el).toHaveClass('now')
  })

  it('shows "--" when caught up and no future cards', () => {
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt: null } })
    expect(screen.getByText('--')).toBeInTheDocument()
  })

  it('shows countdown when caught up and nextDueAt is in the future', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-03T10:00:00Z'))
    const nextDueAt = '2026-06-03T12:30:00Z'
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt } })
    expect(screen.getByText('2h 30m')).toBeInTheDocument()
  })

  it('shows minutes only when under 1 hour', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-03T10:00:00Z'))
    const nextDueAt = '2026-06-03T10:45:00Z'
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt } })
    expect(screen.getByText('45m')).toBeInTheDocument()
  })
})

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import ProgressBreakdown from './ProgressBreakdown.svelte'

describe('ProgressBreakdown', () => {
  it('renders one legend entry per non-zero bucket with counts', () => {
    render(ProgressBreakdown, {
      progress: { not_seen: 10, learning: 5, known: 3, relearning: 0 },
    })
    expect(screen.getByText(/not seen \(10\)/i)).toBeInTheDocument()
    expect(screen.getByText(/learning \(5\)/i)).toBeInTheDocument()
    expect(screen.getByText(/known \(3\)/i)).toBeInTheDocument()
    expect(screen.queryByText(/relearning/i)).not.toBeInTheDocument()
  })

  it('shows the empty state when all buckets are zero', () => {
    render(ProgressBreakdown, {
      progress: { not_seen: 0, learning: 0, known: 0, relearning: 0 },
    })
    expect(screen.getByText(/no cards yet/i)).toBeInTheDocument()
  })
})

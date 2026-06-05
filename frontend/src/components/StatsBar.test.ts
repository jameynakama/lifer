import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import StatsBar from './StatsBar.svelte'

describe('StatsBar', () => {
  it('renders each stat value and label', () => {
    render(StatsBar, {
      props: { stats: [{ label: 'Due today', value: 11 }, { label: 'Streak', value: 5 }] },
    })
    expect(screen.getByText('11')).toBeInTheDocument()
    expect(screen.getByText('Due today')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('Streak')).toBeInTheDocument()
  })

  it('renders nothing when stats is empty', () => {
    const { container } = render(StatsBar, { props: { stats: [] } })
    expect(container.querySelector('.stat')).toBeNull()
  })
})

describe('StatsBar grid variant', () => {
  it('applies the grid variant class and card chrome', () => {
    render(StatsBar, {
      stats: [{ label: 'DUE TODAY', value: 3, highlight: true }],
      variant: 'grid',
    })
    expect(document.querySelector('.stats-bar.grid')).toBeInTheDocument()
    expect(document.querySelector('.stat.card')).toBeInTheDocument()
    expect(document.querySelector('.value.now')).toBeInTheDocument()
  })
})

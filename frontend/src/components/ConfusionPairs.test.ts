import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import ConfusionPairs from './ConfusionPairs.svelte'

const pair = {
  actual: {
    ebird_code: 'foxspa',
    common_name: 'Fox Sparrow',
    scientific_name: 'Passerella iliaca',
  },
  guessed: {
    ebird_code: 'sonspa',
    common_name: 'Song Sparrow',
    scientific_name: 'Melospiza melodia',
  },
  count: 4,
}

describe('ConfusionPairs', () => {
  it('renders actual → guessed with the miss count', () => {
    render(ConfusionPairs, { confusions: [pair] })
    expect(screen.getByText(/fox sparrow/i)).toBeInTheDocument()
    expect(screen.getByText(/song sparrow/i)).toBeInTheDocument()
    expect(screen.getByText(/4×/)).toBeInTheDocument()
  })

  it('shows the empty state when there are no confusions', () => {
    render(ConfusionPairs, { confusions: [] })
    expect(screen.getByText(/no confusions yet/i)).toBeInTheDocument()
  })
})

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import RevealCard from './RevealCard.svelte'

const card = {
  id: '1',
  recording_path: '/recordings/song-sparrow.mp3',
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  photo_path: '/photos/song-sparrow.jpg',
}

describe('RevealCard', () => {
  it('renders the species common and scientific name', () => {
    render(RevealCard, { props: { card, onRate: vi.fn() } })
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Melospiza melodia')).toBeInTheDocument()
  })

  it('renders a species photo', () => {
    render(RevealCard, { props: { card, onRate: vi.fn() } })
    const img = screen.getByRole('img', { name: /song sparrow/i })
    expect(img).toHaveAttribute('src', '/photos/song-sparrow.jpg')
  })

  it('renders four confidence rating buttons', () => {
    render(RevealCard, { props: { card, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /again/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /hard/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /good/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /easy/i })).toBeInTheDocument()
  })

  it('calls onRate with 1 when Again is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /again/i }))
    expect(onRate).toHaveBeenCalledWith(1)
  })

  it('calls onRate with 4 when Easy is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /easy/i }))
    expect(onRate).toHaveBeenCalledWith(4)
  })
})

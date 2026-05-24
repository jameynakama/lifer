import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import RevealCard from './RevealCard.svelte'
import type { BirdCard, Species } from '../types'

const card: BirdCard = {
  species_id: 1,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio' as const,
}

const songSparrow: Species = {
  id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa',
}
const foxSparrow: Species = {
  id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa',
}

describe('RevealCard', () => {
  it('renders the species common and scientific name', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Melospiza melodia')).toBeInTheDocument()
  })

  it('renders a species photo', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    const img = screen.getByRole('img', { name: /song sparrow/i })
    expect(img).toHaveAttribute('src', '/photos/song-sparrow.jpg')
  })

  it('renders four confidence rating buttons', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /again/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /hard/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /good/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /easy/i })).toBeInTheDocument()
  })

  it('calls onRate with 1 when Again is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, correct: false, guessed: null, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /again/i }))
    expect(onRate).toHaveBeenCalledWith(1)
  })

  it('calls onRate with 4 when Easy is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /easy/i }))
    expect(onRate).toHaveBeenCalledWith(4)
  })

  it('shows correct banner when answer is right', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    const banner = screen.getByText(/✓/).closest('.result-banner')
    expect(banner).not.toBeNull()
    expect(banner).toHaveClass('correct')
    expect(banner).not.toHaveClass('incorrect')
    expect(screen.getByText(/✓ Song Sparrow/)).toBeInTheDocument()
  })

  it('shows wrong banner with guessed name when answer is wrong', () => {
    render(RevealCard, { props: { card, correct: false, guessed: foxSparrow, onRate: vi.fn() } })
    const banner = screen.getByText(/✗/).closest('.result-banner')
    expect(banner).not.toBeNull()
    expect(banner).toHaveClass('incorrect')
    expect(banner).not.toHaveClass('correct')
    expect(screen.getByText(/you guessed: fox sparrow/i)).toBeInTheDocument()
  })

  it("shows 'You didn't know' when I don't know was selected", () => {
    render(RevealCard, { props: { card, correct: false, guessed: null, onRate: vi.fn() } })
    expect(screen.getByText(/you didn't know/i)).toBeInTheDocument()
  })

  it('Good button has suggested class when correct', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /good/i })).toHaveClass('suggested')
    expect(screen.getByRole('button', { name: /again/i })).not.toHaveClass('suggested')
  })

  it('Again button has suggested class when incorrect', () => {
    render(RevealCard, { props: { card, correct: false, guessed: null, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /again/i })).toHaveClass('suggested')
    expect(screen.getByRole('button', { name: /good/i })).not.toHaveClass('suggested')
  })
})

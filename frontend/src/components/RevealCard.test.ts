import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import RevealCard from './RevealCard.svelte'
import type { BirdCard, Species } from '../types'

vi.mock('wavesurfer.js', () => ({
  default: {
    create: vi.fn(() => ({
      on: vi.fn((event: string, cb: () => void) => { if (event === 'ready') cb() }),
      pause: vi.fn(),
      destroy: vi.fn(),
    })),
  },
}))

const card: BirdCard = {
  ebird_code: 'sonspa',
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio' as const,
  recording_type: 'song',
  recording_credit: '',
  photo_credit: '',
  due_remaining: 5,
}

const songSparrow: Species = {
  common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa',
}
const foxSparrow: Species = {
  common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa',
}

describe('RevealCard', () => {
  it('renders the species common and scientific name', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onNext: vi.fn() } })
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Melospiza melodia')).toBeInTheDocument()
  })

  it('renders a species photo', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onNext: vi.fn() } })
    const img = screen.getByRole('img', { name: /song sparrow/i })
    expect(img).toHaveAttribute('src', '/photos/song-sparrow.jpg')
  })

  it('renders a Next button', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onNext: vi.fn() } })
    expect(screen.getByRole('button', { name: /next/i })).toBeInTheDocument()
  })

  it('calls onNext when Next is clicked', async () => {
    const onNext = vi.fn()
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onNext } })
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    expect(onNext).toHaveBeenCalled()
  })

  it('shows correct banner when answer is right', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onNext: vi.fn() } })
    const banner = screen.getByText(/✓/).closest('.result-banner')
    expect(banner).not.toBeNull()
    expect(banner).toHaveClass('correct')
    expect(banner).not.toHaveClass('incorrect')
    expect(screen.getByText(/✓ Song Sparrow/)).toBeInTheDocument()
  })

  it('shows wrong banner with guessed name when answer is wrong', () => {
    render(RevealCard, { props: { card, correct: false, guessed: foxSparrow, onNext: vi.fn() } })
    const banner = screen.getByText(/✗/).closest('.result-banner')
    expect(banner).not.toBeNull()
    expect(banner).toHaveClass('incorrect')
    expect(banner).not.toHaveClass('correct')
    expect(screen.getByText(/you guessed: fox sparrow/i)).toBeInTheDocument()
  })

  it("shows 'You didn't know' when I don't know was selected", () => {
    render(RevealCard, { props: { card, correct: false, guessed: null, onNext: vi.fn() } })
    expect(screen.getByText(/you didn't know/i)).toBeInTheDocument()
  })
})

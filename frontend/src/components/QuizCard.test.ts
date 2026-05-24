import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import WaveSurfer from 'wavesurfer.js'
import QuizCard from './QuizCard.svelte'
import type { BirdCard, Species } from '../types'

// WaveSurfer uses Web Audio API not available in jsdom -- mock the whole thing
const mockWs = {
  on: vi.fn((event: string, cb: () => void) => {
    if (event === 'ready') cb()
  }),
  playPause: vi.fn(),
  destroy: vi.fn(),
}

vi.mock('wavesurfer.js', () => ({
  default: { create: vi.fn(() => mockWs) },
}))

const card: BirdCard = {
  species_id: 1,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio' as const,
}

const species: Species[] = [
  { id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
]

describe('QuizCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockWs.on.mockImplementation((event: string, cb: () => void) => {
      if (event === 'ready') cb()
    })
  })

  it('initialises WaveSurfer with the media url', () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    expect(vi.mocked(WaveSurfer.create)).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/recordings/song-sparrow.mp3' })
    )
  })

  it('Reveal button is disabled initially', () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    expect(screen.getByRole('button', { name: /reveal answer/i })).toBeDisabled()
  })

  it('Reveal button is enabled after a species is selected from the typeahead', async () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    const input = screen.getByRole('combobox')
    await fireEvent.input(input, { target: { value: 'song' } })
    const option = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(option)
    expect(screen.getByRole('button', { name: /reveal answer/i })).not.toBeDisabled()
  })

  it('calls onReveal with the selected species when Reveal is clicked', async () => {
    const onReveal = vi.fn()
    render(QuizCard, { props: { card, species, onReveal } })
    const input = screen.getByRole('combobox')
    await fireEvent.input(input, { target: { value: 'song' } })
    await fireEvent.mouseDown(screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(onReveal).toHaveBeenCalledWith(species[0])
  })

  it("calls onReveal(null) when I don't know is clicked", async () => {
    const onReveal = vi.fn()
    render(QuizCard, { props: { card, species, onReveal } })
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    expect(onReveal).toHaveBeenCalledWith(null)
  })
})

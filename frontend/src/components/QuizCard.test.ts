import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import QuizCard from './QuizCard.svelte'
import type { BirdCard, Species } from '../types'

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
  it('renders an audio player with the media url', () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    const audio = document.querySelector('audio')
    expect(audio).not.toBeNull()
    expect(audio!.src).toContain('/recordings/song-sparrow.mp3')
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

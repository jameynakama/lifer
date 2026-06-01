import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import ImageQuizCard from './ImageQuizCard.svelte'
import type { BirdCard, Species } from '../types'

const card: BirdCard = {
  ebird_code: 'sonspa',
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/photos/song-sparrow.jpg',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'image' as const,
  recording_type: '',
  due_remaining: 5,
}

const species: Species[] = [
  { common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
]

describe('ImageQuizCard', () => {
  it('renders an image with the media url', () => {
    render(ImageQuizCard, { props: { card, species, onReveal: vi.fn() } })
    const img = document.querySelector('img.quiz-photo')
    expect(img).not.toBeNull()
    expect(img!.getAttribute('src')).toBe('/photos/song-sparrow.jpg')
  })

  it('Reveal button is disabled initially', () => {
    render(ImageQuizCard, { props: { card, species, onReveal: vi.fn() } })
    expect(screen.getByRole('button', { name: /reveal answer/i })).toBeDisabled()
  })

  it('Reveal button is enabled after a species is selected from the typeahead', async () => {
    render(ImageQuizCard, { props: { card, species, onReveal: vi.fn() } })
    const input = screen.getByRole('combobox')
    await fireEvent.input(input, { target: { value: 'song' } })
    const option = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(option)
    expect(screen.getByRole('button', { name: /reveal answer/i })).not.toBeDisabled()
  })

  it('calls onReveal with the selected species when Reveal is clicked', async () => {
    const onReveal = vi.fn()
    render(ImageQuizCard, { props: { card, species, onReveal } })
    const input = screen.getByRole('combobox')
    await fireEvent.input(input, { target: { value: 'song' } })
    await fireEvent.mouseDown(screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(onReveal).toHaveBeenCalledWith(species[0])
  })

  it("calls onReveal(null) when I don't know is clicked", async () => {
    const onReveal = vi.fn()
    render(ImageQuizCard, { props: { card, species, onReveal } })
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    expect(onReveal).toHaveBeenCalledWith(null)
  })
})

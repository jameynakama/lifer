import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import QuizCard from './QuizCard.svelte'

const card = {
  species_id: 1,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio' as const,
}

describe('QuizCard', () => {
  it('renders an audio player with the media url', () => {
    render(QuizCard, { props: { card, onReveal: vi.fn() } })
    const audio = document.querySelector('audio')
    expect(audio).not.toBeNull()
    expect(audio!.src).toContain('/recordings/song-sparrow.mp3')
  })

  it('renders a text input for the species guess', () => {
    render(QuizCard, { props: { card, onReveal: vi.fn() } })
    expect(screen.getByPlaceholderText(/type species name/i)).toBeInTheDocument()
  })

  it('calls onReveal when Reveal Answer is clicked', async () => {
    const onReveal = vi.fn()
    render(QuizCard, { props: { card, onReveal } })
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(onReveal).toHaveBeenCalledOnce()
  })
})

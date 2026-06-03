import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import DeckList from './DeckList.svelte'

const decks = [
  { id: 1, name: 'Pacific Northwest', audio_due: 8, image_due: 5 },
  { id: 2, name: 'My Warblers', audio_due: 3, image_due: 0 },
]

describe('DeckList', () => {
  it('renders each deck name', () => {
    render(DeckList, { props: { decks, onPractice: vi.fn() } })
    expect(screen.getByText('Pacific Northwest')).toBeInTheDocument()
    expect(screen.getByText('My Warblers')).toBeInTheDocument()
  })

  it('shows audio button when audio_due > 0', () => {
    render(DeckList, { props: { decks, onPractice: vi.fn() } })
    const audioButtons = screen.getAllByRole('button', { name: /audio/i })
    expect(audioButtons.length).toBeGreaterThan(0)
  })

  it('shows image button when image_due > 0', () => {
    render(DeckList, { props: { decks, onPractice: vi.fn() } })
    const imageButtons = screen.getAllByRole('button', { name: /image/i })
    expect(imageButtons.length).toBeGreaterThan(0)
  })

  it('shows All done when both lanes are 0', () => {
    const noDueDecks = [{ id: 3, name: 'Empty Deck', audio_due: 0, image_due: 0 }]
    render(DeckList, { props: { decks: noDueDecks, onPractice: vi.fn() } })
    expect(screen.getByText('All done')).toBeInTheDocument()
  })

  it('calls onPractice with deck and audio lane when audio button clicked', async () => {
    const onPractice = vi.fn()
    render(DeckList, { props: { decks, onPractice } })
    await fireEvent.click(screen.getAllByRole('button', { name: /audio/i })[0])
    expect(onPractice).toHaveBeenCalledWith(decks[0], 'audio')
  })

  it('calls onPractice with deck and image lane when image button clicked', async () => {
    const onPractice = vi.fn()
    render(DeckList, { props: { decks, onPractice } })
    await fireEvent.click(screen.getAllByRole('button', { name: /image/i })[0])
    expect(onPractice).toHaveBeenCalledWith(decks[0], 'image')
  })
})

import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from '../stores/view'
import Quiz from './Quiz.svelte'

beforeEach(() => {
  view.set('quiz')
})

describe('Quiz', () => {
  it('shows QuizCard initially (audio player visible, no photo)', () => {
    render(Quiz)
    expect(document.querySelector('audio')).not.toBeNull()
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('shows RevealCard after Reveal Answer is clicked', async () => {
    render(Quiz)
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(screen.getByRole('img')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /again/i })).toBeInTheDocument()
  })

  it('advances to the next card after rating', async () => {
    render(Quiz)
    const firstSrc = document.querySelector('audio')!.src
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    await fireEvent.click(screen.getByRole('button', { name: /good/i }))
    expect(document.querySelector('audio')!.src).not.toBe(firstSrc)
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('navigates to dashboard when all cards are rated', async () => {
    render(Quiz)
    const rateAll = async () => {
      await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
      await fireEvent.click(screen.getByRole('button', { name: /good/i }))
    }
    // MOCK_CARDS has 2 cards
    await rateAll()
    await rateAll()
    expect(get(view)).toBe('dashboard')
  })
})

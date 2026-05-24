import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from '../stores/view'
import Quiz from './Quiz.svelte'

const card = {
  species_id: 99,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio',
}

beforeEach(() => {
  view.set('quiz')
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Quiz', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(Quiz, { props: { groupId: '1', lane: 'audio' } })
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('shows QuizCard when a card is returned', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve(card),
    }))
    render(Quiz, { props: { groupId: '1', lane: 'audio' } })
    await vi.waitFor(() => {
      expect(document.querySelector('audio')).not.toBeNull()
    })
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('shows RevealCard after Reveal Answer is clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve(card),
    }))
    render(Quiz, { props: { groupId: '1', lane: 'audio' } })
    await vi.waitFor(() => screen.getByRole('button', { name: /reveal answer/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(screen.getByRole('img')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /again/i })).toBeInTheDocument()
  })

  it('shows All done when 204 is returned', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    render(Quiz, { props: { groupId: '1', lane: 'audio' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/all done/i)).toBeInTheDocument()
    })
  })

  it('navigates to dashboard when All done button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    render(Quiz, { props: { groupId: '1', lane: 'audio' } })
    await vi.waitFor(() => screen.getByRole('button', { name: /back to dashboard/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to dashboard/i }))
    expect(get(view)).toBe('dashboard')
  })
})

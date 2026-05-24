import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import QuizPage from './+page.svelte'

const card = {
  species_id: 99,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio',
}

beforeEach(() => {
  page.params = { id: '42' }
  page.url = new URL('http://localhost/groups/42/quiz?lane=audio')
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Quiz page', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(QuizPage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('shows QuizCard when a card is returned', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve(card),
    }))
    render(QuizPage)
    await vi.waitFor(() => {
      expect(document.querySelector('audio')).not.toBeNull()
    })
  })

  it('shows All done when 204 is returned', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/all done/i)).toBeInTheDocument()
    })
  })

  it('navigates to group detail when All done button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /back to group/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to group/i }))
    expect(goto).toHaveBeenCalledWith('/groups/42')
  })
})

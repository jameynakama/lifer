import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/svelte'
import { renderWithClient } from '../test-utils'
import DeckPickerPopover from './DeckPickerPopover.svelte'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const decks = [
  { id: 1, name: 'My Warblers', audio_due: 0, image_due: 0 },
  { id: 2, name: 'Sparrows', audio_due: 0, image_due: 0 },
]

describe('DeckPickerPopover', () => {
  it('shows a list of user decks', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ decks, next_due_at: null }),
      }),
    )
    renderWithClient(DeckPickerPopover, { onPick: vi.fn(), onClose: vi.fn() })
    await vi.waitFor(() => {
      expect(screen.getByText(/my warblers/i)).toBeInTheDocument()
      expect(screen.getByText(/sparrows/i)).toBeInTheDocument()
    })
  })

  it('calls onPick with deck id when a deck is selected', async () => {
    const onPick = vi.fn()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ decks, next_due_at: null }),
      }),
    )
    renderWithClient(DeckPickerPopover, { onPick, onClose: vi.fn() })
    await vi.waitFor(() => screen.getByText(/my warblers/i))
    await fireEvent.click(screen.getByText(/my warblers/i))
    expect(onPick).toHaveBeenCalledWith(1)
  })

  it('creates a new deck and calls onPick with the new deck id', async () => {
    const onPick = vi.fn()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
        if (opts?.method === 'POST') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ id: 99, name: 'Sparrows' }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ decks: [], next_due_at: null }),
        })
      }),
    )
    renderWithClient(DeckPickerPopover, { onPick, onClose: vi.fn() })
    await vi.waitFor(() => screen.getByPlaceholderText(/new deck name/i))
    await fireEvent.input(screen.getByPlaceholderText(/new deck name/i), {
      target: { value: 'Sparrows' },
    })
    await fireEvent.click(screen.getByRole('button', { name: /create/i }))
    await vi.waitFor(() => {
      expect(onPick).toHaveBeenCalledWith(99)
    })
  })

  it('calls onClose when backdrop clicked', async () => {
    const onClose = vi.fn()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ decks: [], next_due_at: null }),
      }),
    )
    renderWithClient(DeckPickerPopover, { onPick: vi.fn(), onClose })
    await vi.waitFor(() => screen.getByRole('dialog'))
    await fireEvent.click(screen.getByTestId('backdrop'))
    expect(onClose).toHaveBeenCalledOnce()
  })
})

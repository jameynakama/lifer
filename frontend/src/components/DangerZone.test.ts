import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/svelte'
import { renderWithClient } from '../test-utils'
import DangerZone from './DangerZone.svelte'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

function okFetch(body = { cards_deleted: 412, reviews_deleted: 1038 }) {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve(body),
  })
}

/** Click a row's arm button, type the confirm text, return the live confirm button (or null if strip closed). */
async function armRow(name: RegExp, text?: string) {
  await fireEvent.click(screen.getByRole('button', { name }))
  if (text !== undefined) {
    await fireEvent.input(screen.getByRole('textbox', { name: /type reset to confirm/i }), {
      target: { value: text },
    })
  }
  // Use query (not get) so toggle-close calls don't throw when the strip unmounts.
  return screen.queryByRole('button', { name: /confirm/i }) as HTMLElement
}

describe('DangerZone', () => {
  it('renders both reset rows with their copy', () => {
    vi.stubGlobal('fetch', okFetch())
    renderWithClient(DangerZone)
    expect(screen.getByText(/danger zone/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /reset schedule/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /reset everything/i })).toBeInTheDocument()
    expect(screen.getByText(/review history and stats survive/i)).toBeInTheDocument()
    expect(screen.getByText(/clean slate/i)).toBeInTheDocument()
  })

  it('keeps confirm disabled until the input is exactly RESET', async () => {
    vi.stubGlobal('fetch', okFetch())
    renderWithClient(DangerZone)

    const confirm = await armRow(/reset schedule/i)
    expect(confirm).toBeDisabled()

    await armRow(/reset schedule/i) // toggles closed
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()

    await fireEvent.click(screen.getByRole('button', { name: /reset schedule/i }))
    await fireEvent.input(screen.getByRole('textbox', { name: /type reset to confirm/i }), {
      target: { value: 'reset' },
    })
    expect(screen.getByRole('button', { name: /confirm/i })).toBeDisabled()

    await fireEvent.input(screen.getByRole('textbox', { name: /type reset to confirm/i }), {
      target: { value: 'RESET' },
    })
    expect(screen.getByRole('button', { name: /confirm/i })).toBeEnabled()
  })

  it('opening one row closes the other', async () => {
    vi.stubGlobal('fetch', okFetch())
    renderWithClient(DangerZone)
    await fireEvent.click(screen.getByRole('button', { name: /reset schedule/i }))
    expect(screen.getAllByRole('textbox')).toHaveLength(1)
    await fireEvent.click(screen.getByRole('button', { name: /reset everything/i }))
    expect(screen.getAllByRole('textbox')).toHaveLength(1)
  })

  it('POSTs scope schedule and shows the cards count', async () => {
    const fetchMock = okFetch({ cards_deleted: 412, reviews_deleted: 0 })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(DangerZone)

    const confirm = await armRow(/reset schedule/i, 'RESET')
    await fireEvent.click(confirm)

    await vi.waitFor(() => {
      expect(screen.getByText(/deleted 412 cards\./i)).toBeInTheDocument()
    })
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/reset')
    expect(JSON.parse(init.body as string)).toEqual({ scope: 'schedule' })
  })

  it('POSTs scope everything, shows both counts, and invalidates decks + stats', async () => {
    vi.stubGlobal('fetch', okFetch())
    const { client } = renderWithClient(DangerZone)
    const invalidate = vi.spyOn(client, 'invalidateQueries')

    const confirm = await armRow(/reset everything/i, 'RESET')
    await fireEvent.click(confirm)

    await vi.waitFor(() => {
      expect(screen.getByText(/deleted 412 cards and 1,038 reviews\./i)).toBeInTheDocument()
    })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['decks'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['stats'] })
  })

  it('disables confirm while the reset is in flight', async () => {
    let resolveFetch!: (v: unknown) => void
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise((r) => (resolveFetch = r))))
    renderWithClient(DangerZone)

    const confirm = await armRow(/reset everything/i, 'RESET')
    await fireEvent.click(confirm)

    expect(screen.getByRole('button', { name: /confirm/i })).toBeDisabled()

    resolveFetch({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ cards_deleted: 1, reviews_deleted: 2 }),
    })
    await vi.waitFor(() => {
      expect(screen.getByText(/deleted 1 cards and 2 reviews\./i)).toBeInTheDocument()
    })
  })

  it('shows an error and skips invalidation when the reset fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, text: () => Promise.resolve('') }),
    )
    const { client } = renderWithClient(DangerZone)
    const invalidate = vi.spyOn(client, 'invalidateQueries')

    const confirm = await armRow(/reset everything/i, 'RESET')
    await fireEvent.click(confirm)

    await vi.waitFor(() => {
      expect(screen.getByText(/reset failed/i)).toBeInTheDocument()
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { page } from '$app/state'
import Page from './+page.svelte'

beforeEach(() => {
  page.params = { ebird_code: 'amro' }
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Admin species detail page', () => {
  it('renders images and recordings sections', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            images: [
              {
                macaulay_id: 'img1',
                file_path: 'https://example.com/img1.jpg',
                credit: 'Photographer',
              },
            ],
            recordings: [
              {
                xeno_canto_id: 'rec1',
                file_path: 'https://example.com/rec1.mp3',
                quality: 'A',
                type: 'song',
              },
            ],
          }),
      }),
    )
    render(Page)
    await vi.waitFor(() => {
      expect(screen.getByText(/images/i)).toBeTruthy()
      expect(screen.getByText(/recordings/i)).toBeTruthy()
    })
  })
})

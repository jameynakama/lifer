import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render } from '@testing-library/svelte'
import WavePlayer from './WavePlayer.svelte'

// WaveSurfer uses Web Audio API not available in jsdom -- mock the whole thing.
// Wrapped in a spy so we can inspect the options passed to create().
const mockWs = {
  on: vi.fn((event: string, cb: () => void) => {
    if (event === 'ready') cb()
  }),
  playPause: vi.fn(),
  pause: vi.fn(),
  destroy: vi.fn(),
}

const create = vi.fn((..._args: unknown[]) => mockWs)

vi.mock('wavesurfer.js', () => ({
  default: { create: (...args: unknown[]) => create(...args) },
}))

describe('WavePlayer', () => {
  beforeEach(() => create.mockClear())

  it('draws the supplied peaks, scaled to 0..1', () => {
    render(WavePlayer, { url: '/x.mp3', peaks: [0, 255, 128] })

    const opts = create.mock.calls[0][0] as { peaks: number[][] }
    expect(opts.peaks).toHaveLength(1)
    expect(opts.peaks[0][0]).toBeCloseTo(0, 3)
    expect(opts.peaks[0][1]).toBeCloseTo(1, 3)
    expect(opts.peaks[0][2]).toBeCloseTo(0.502, 2)
  })

  it('falls back to generated peaks when none are supplied', () => {
    render(WavePlayer, { url: '/x.mp3' })

    const opts = create.mock.calls[0][0] as { peaks: number[][] }
    expect(opts.peaks[0]).toHaveLength(200)
    // Generated bars are all positive; they exist only so the bars draw
    // instantly for recordings the backfill has not reached.
    expect(Math.min(...opts.peaks[0])).toBeGreaterThan(0)
  })

  it('falls back when peaks is an empty array', () => {
    render(WavePlayer, { url: '/x.mp3', peaks: [] })

    const opts = create.mock.calls[0][0] as { peaks: number[][] }
    expect(opts.peaks[0]).toHaveLength(200)
  })
})

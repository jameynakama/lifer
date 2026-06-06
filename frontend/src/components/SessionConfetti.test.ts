import { vi, describe, it, expect, afterEach } from 'vitest'
import { render } from '@testing-library/svelte'
import confetti from 'canvas-confetti'
import SessionConfetti from './SessionConfetti.svelte'

describe('SessionConfetti', () => {
  afterEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('fires two confetti bursts on mount', () => {
    vi.useFakeTimers()
    render(SessionConfetti)
    vi.runAllTimers()
    expect(vi.mocked(confetti)).toHaveBeenCalledTimes(2)
  })

  it('uses feather and bird shapes', () => {
    vi.useFakeTimers()
    render(SessionConfetti)
    vi.runAllTimers()
    expect(vi.mocked(confetti.shapeFromText)).toHaveBeenCalledWith(
      expect.objectContaining({ text: '🪶' }),
    )
    expect(vi.mocked(confetti.shapeFromText)).toHaveBeenCalledWith(
      expect.objectContaining({ text: '🐦' }),
    )
  })
})

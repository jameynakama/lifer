import '@testing-library/jest-dom'
import { vi } from 'vitest'

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({ matches: false, media: query }),
})

window.HTMLMediaElement.prototype.pause = () => {}
window.HTMLMediaElement.prototype.play = () => Promise.resolve()

vi.mock('canvas-confetti', () => ({
  default: Object.assign(vi.fn(), {
    shapeFromText: vi.fn().mockReturnValue({ type: 'text' }),
  }),
}))

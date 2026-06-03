import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import InstallPrompt from './InstallPrompt.svelte'

const originalUA = navigator.userAgent

function setUserAgent(ua: string) {
  Object.defineProperty(navigator, 'userAgent', { value: ua, configurable: true })
}

function setStandalone(standalone: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockReturnValue({ matches: standalone }),
  })
}

beforeEach(() => {
  localStorage.clear()
  setStandalone(false)
})

afterEach(() => {
  Object.defineProperty(navigator, 'userAgent', { value: originalUA, configurable: true })
})

const ANDROID_UA = 'Mozilla/5.0 (Linux; Android 12; Pixel 6) AppleWebKit/537.36'
const IOS_UA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15'
const DESKTOP_UA = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'

describe('InstallPrompt', () => {
  it('shows Android instructions on Android', () => {
    setUserAgent(ANDROID_UA)
    render(InstallPrompt)
    expect(screen.getByText(/tap ⋮/i)).toBeInTheDocument()
  })

  it('shows iOS instructions on iPhone', () => {
    setUserAgent(IOS_UA)
    render(InstallPrompt)
    expect(screen.getByText(/share button/i)).toBeInTheDocument()
  })

  it('shows nothing on desktop', () => {
    setUserAgent(DESKTOP_UA)
    render(InstallPrompt)
    expect(screen.queryByText(/home screen/i)).not.toBeInTheDocument()
  })

  it('shows nothing when already in standalone mode', () => {
    setUserAgent(ANDROID_UA)
    setStandalone(true)
    render(InstallPrompt)
    expect(screen.queryByText(/home screen/i)).not.toBeInTheDocument()
  })

  it('shows nothing when previously dismissed', () => {
    setUserAgent(IOS_UA)
    localStorage.setItem('dismissed-install-prompt', '1')
    render(InstallPrompt)
    expect(screen.queryByText(/home screen/i)).not.toBeInTheDocument()
  })

  it('hides banner and sets localStorage when dismissed', async () => {
    setUserAgent(ANDROID_UA)
    render(InstallPrompt)
    expect(screen.getByText(/tap ⋮/i)).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: /dismiss/i }))
    expect(screen.queryByText(/tap ⋮/i)).not.toBeInTheDocument()
    expect(localStorage.getItem('dismissed-install-prompt')).toBe('1')
  })
})

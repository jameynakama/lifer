import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import InfoTip from './InfoTip.svelte'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('InfoTip', () => {
  it('renders a labelled button and no tooltip initially', () => {
    render(InfoTip, { props: { text: 'Explains the panel.' } })
    expect(screen.getByRole('button', { name: /what does this panel show/i })).toBeInTheDocument()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('toggles the tooltip on click and reflects aria-expanded', async () => {
    render(InfoTip, { props: { text: 'Explains the panel.' } })
    const btn = screen.getByRole('button', { name: /what does this panel show/i })
    expect(btn).toHaveAttribute('aria-expanded', 'false')

    await fireEvent.click(btn)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Explains the panel.')
    expect(btn).toHaveAttribute('aria-expanded', 'true')

    await fireEvent.click(btn)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(btn).toHaveAttribute('aria-expanded', 'false')
  })

  it('closes on Escape', async () => {
    render(InfoTip, { props: { text: 'Explains the panel.' } })
    await fireEvent.click(screen.getByRole('button', { name: /what does this panel show/i }))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()

    await fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('closes on a click outside, stays open on a click inside', async () => {
    render(InfoTip, { props: { text: 'Explains the panel.' } })
    await fireEvent.click(screen.getByRole('button', { name: /what does this panel show/i }))

    await fireEvent.click(screen.getByRole('tooltip'))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()

    await fireEvent.click(document.body)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})

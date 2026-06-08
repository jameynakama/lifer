import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import About from './+page.svelte'

describe('About page', () => {
  it('renders the major sections in priority order', () => {
    render(About)
    const headings = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
    expect(headings).toEqual([
      'What FlockDeck is',
      'Getting started',
      'Audio & image lanes',
      'The practice loop',
      'How scheduling works',
      'Credits',
    ])
  })

  it('exposes a #getting-started anchor for the homepage link', () => {
    const { container } = render(About)
    expect(container.querySelector('#getting-started')).not.toBeNull()
  })

  it('links contact to the plus-aliased email address (frontloaded + in credits)', () => {
    const { container } = render(About)
    const mailto = container.querySelectorAll('a[href="mailto:nakamajamey+flockdeck@gmail.com"]')
    expect(mailto.length).toBeGreaterThanOrEqual(2)
  })

  it('credits the three data sources', () => {
    render(About)
    // eBird is mentioned both in "Getting started" and in the credits list,
    // so getByText would throw on the duplicate; getAllByText tolerates it.
    expect(screen.getAllByText(/ebird/i).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/xeno-canto/i)).toBeInTheDocument()
    expect(screen.getByText(/macaulay/i)).toBeInTheDocument()
  })
})

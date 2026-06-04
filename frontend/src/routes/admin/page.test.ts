import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import Page from './+page.svelte'

describe('Admin landing page', () => {
  it('renders a link to species admin', () => {
    render(Page)
    const link = screen.getByRole('link', { name: /species/i })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', '/admin/species')
  })

  it('renders a link to preset decks admin', () => {
    render(Page)
    const link = screen.getByRole('link', { name: /preset decks/i })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', '/admin/decks')
  })
})

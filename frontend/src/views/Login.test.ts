import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import Login from './Login.svelte'

describe('Login', () => {
  it('renders a link to the Google auth endpoint', () => {
    render(Login)
    const link = screen.getByRole('link', { name: /sign in with google/i })
    expect(link).toHaveAttribute('href', '/api/v1/auth/google')
  })
})

describe('Login auth error', () => {
  it('shows a sign-in error message when the URL carries ?error=auth_state', () => {
    const url = new URL('http://localhost/?error=auth_state')
    vi.stubGlobal('location', { ...window.location, search: url.search })
    render(Login)
    expect(screen.getByText(/sign-in didn.t complete/i)).toBeInTheDocument()
  })

  it('shows no error message normally', () => {
    vi.stubGlobal('location', { ...window.location, search: '' })
    render(Login)
    expect(screen.queryByText(/sign-in didn.t complete/i)).toBeNull()
  })
})

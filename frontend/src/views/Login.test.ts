import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import Login from './Login.svelte'

describe('Login', () => {
  it('renders a link to the Google auth endpoint', () => {
    render(Login)
    const link = screen.getByRole('link', { name: /sign in with google/i })
    expect(link).toHaveAttribute('href', '/api/v1/auth/google')
  })
})

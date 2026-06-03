import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { auth } from '$stores/auth'
import Layout from './+layout.svelte'

beforeEach(() => {
  auth.set(null)
  vi.mocked(goto).mockClear()
})

describe('Admin Layout', () => {
  it('redirects non-admin users to /', async () => {
    auth.set({ id: 1, email: 'test@example.com', name: 'Test', is_admin: false })
    render(Layout)
    await vi.waitFor(() => {
      expect(goto).toHaveBeenCalledWith('/')
    })
  })

  it('redirects unauthenticated users to /', async () => {
    auth.set(null)
    render(Layout)
    await vi.waitFor(() => {
      expect(goto).toHaveBeenCalledWith('/')
    })
  })

  it('renders children for admin users', async () => {
    auth.set({ id: 1, email: 'admin@example.com', name: 'Admin', is_admin: true })
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByText(/admin/i)).toBeInTheDocument()
    })
  })
})

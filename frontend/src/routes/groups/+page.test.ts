import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import GroupsPage from './+page.svelte'

const groups = [
  { id: 1, name: 'My Warblers', is_preset: false, audio_due: 2, image_due: 0 },
]

beforeEach(() => {
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Groups page', () => {
  it('renders group list from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(groups),
    }))
    render(GroupsPage)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
    })
  })

  it('creates a group when form submitted', async () => {
    let postCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'POST') {
        postCalled = true
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 2, name: 'New Group', is_preset: false, audio_due: 0, image_due: 0 }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(groups) })
    }))
    render(GroupsPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/group name/i))
    await fireEvent.input(screen.getByPlaceholderText(/group name/i), { target: { value: 'New Group' } })
    await fireEvent.click(screen.getByRole('button', { name: /create/i }))
    await vi.waitFor(() => { expect(postCalled).toBe(true) })
    await vi.waitFor(() => {
      expect(screen.queryByText(/new group/i)).not.toBeNull()
    })
  })

  it('navigates to group detail when name clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(groups),
    }))
    render(GroupsPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /my warblers/i }))
    // Link navigates via href, not goto -- just verify the link exists with correct href
    const link = screen.getByRole('link', { name: /my warblers/i })
    expect(link).toHaveAttribute('href', '/groups/1')
  })

  it('deletes a group when delete button clicked', async () => {
    let deleteCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'DELETE') {
        deleteCalled = true
        return Promise.resolve({ ok: true })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(groups) })
    }))
    render(GroupsPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /delete/i }))
    await fireEvent.click(screen.getByRole('button', { name: /delete/i }))
    await vi.waitFor(() => { expect(deleteCalled).toBe(true) })
    await vi.waitFor(() => {
      expect(screen.queryByText(/my warblers/i)).toBeNull()
    })
  })
})

import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import BulkAddBar from './BulkAddBar.svelte'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('BulkAddBar', () => {
  it('renders nothing when selectedCount is 0', () => {
    render(BulkAddBar, { selectedCount: 0, onAdd: vi.fn(), onClear: vi.fn() })
    expect(screen.queryByRole('toolbar')).toBeNull()
  })

  it('shows count and action buttons when selectedCount > 0', () => {
    render(BulkAddBar, { selectedCount: 3, onAdd: vi.fn(), onClear: vi.fn() })
    expect(screen.getByRole('toolbar')).toBeInTheDocument()
    expect(screen.getByText(/3 selected/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /add 3 species/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /clear/i })).toBeInTheDocument()
  })

  it('calls onAdd when add button clicked', async () => {
    const onAdd = vi.fn()
    render(BulkAddBar, { selectedCount: 2, onAdd, onClear: vi.fn() })
    await fireEvent.click(screen.getByRole('button', { name: /add/i }))
    expect(onAdd).toHaveBeenCalledOnce()
  })

  it('calls onClear when clear button clicked', async () => {
    const onClear = vi.fn()
    render(BulkAddBar, { selectedCount: 2, onAdd: vi.fn(), onClear })
    await fireEvent.click(screen.getByRole('button', { name: /clear/i }))
    expect(onClear).toHaveBeenCalledOnce()
  })
})

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import GroupList from './GroupList.svelte'

const groups = [
  { id: '1', name: 'Pacific Northwest', is_preset: true, due_count: 8 },
  { id: '2', name: 'My Warblers', is_preset: false, due_count: 3 },
]

describe('GroupList', () => {
  it('renders each group name and due count', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    expect(screen.getByText('Pacific Northwest')).toBeInTheDocument()
    expect(screen.getByText('8 due')).toBeInTheDocument()
    expect(screen.getByText('My Warblers')).toBeInTheDocument()
    expect(screen.getByText('3 due')).toBeInTheDocument()
  })

  it('renders a Practice button for each group', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    expect(screen.getAllByRole('button', { name: /practice/i })).toHaveLength(2)
  })

  it('calls onPractice with the group when its button is clicked', async () => {
    const onPractice = vi.fn()
    render(GroupList, { props: { groups, onPractice } })
    await fireEvent.click(screen.getAllByRole('button', { name: /practice/i })[0])
    expect(onPractice).toHaveBeenCalledWith(groups[0])
  })
})

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import GroupList from './GroupList.svelte'

const groups = [
  { id: '1', name: 'Pacific Northwest', is_preset: true, audio_due: 8, image_due: 5 },
  { id: '2', name: 'My Warblers', is_preset: false, audio_due: 3, image_due: 0 },
]

describe('GroupList', () => {
  it('renders each group name', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    expect(screen.getByText('Pacific Northwest')).toBeInTheDocument()
    expect(screen.getByText('My Warblers')).toBeInTheDocument()
  })

  it('shows audio button when audio_due > 0', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    const audioButtons = screen.getAllByRole('button', { name: /audio/i })
    expect(audioButtons.length).toBeGreaterThan(0)
  })

  it('shows image button when image_due > 0', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    const imageButtons = screen.getAllByRole('button', { name: /image/i })
    expect(imageButtons.length).toBeGreaterThan(0)
  })

  it('shows All done when both lanes are 0', () => {
    const noDueGroups = [{ id: '3', name: 'Empty Group', is_preset: false, audio_due: 0, image_due: 0 }]
    render(GroupList, { props: { groups: noDueGroups, onPractice: vi.fn() } })
    expect(screen.getByText('All done')).toBeInTheDocument()
  })

  it('calls onPractice with group and audio lane when audio button clicked', async () => {
    const onPractice = vi.fn()
    render(GroupList, { props: { groups, onPractice } })
    await fireEvent.click(screen.getAllByRole('button', { name: /audio/i })[0])
    expect(onPractice).toHaveBeenCalledWith(groups[0], 'audio')
  })

  it('calls onPractice with group and image lane when image button clicked', async () => {
    const onPractice = vi.fn()
    render(GroupList, { props: { groups, onPractice } })
    await fireEvent.click(screen.getAllByRole('button', { name: /image/i })[0])
    expect(onPractice).toHaveBeenCalledWith(groups[0], 'image')
  })
})

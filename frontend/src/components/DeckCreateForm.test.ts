import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import DeckCreateForm from './DeckCreateForm.svelte'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('DeckCreateForm', () => {
  it('disables the button until a non-blank name is typed', async () => {
    render(DeckCreateForm, { label: '+ Create deck', onCreate: vi.fn() })
    const button = screen.getByRole('button', { name: /create deck/i })
    expect(button).toBeDisabled()
    await fireEvent.input(screen.getByPlaceholderText(/new deck name/i), {
      target: { value: '   ' },
    })
    expect(button).toBeDisabled()
    await fireEvent.input(screen.getByPlaceholderText(/new deck name/i), {
      target: { value: 'Warblers' },
    })
    expect(button).not.toBeDisabled()
  })

  it('calls onCreate with the trimmed name on click', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    render(DeckCreateForm, { label: '+ Create deck', onCreate })
    await fireEvent.input(screen.getByPlaceholderText(/new deck name/i), {
      target: { value: '  Warblers  ' },
    })
    await fireEvent.click(screen.getByRole('button', { name: /create deck/i }))
    expect(onCreate).toHaveBeenCalledWith('Warblers')
  })

  it('calls onCreate when Enter is pressed in the input', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    render(DeckCreateForm, { label: '+ Create deck', onCreate })
    const input = screen.getByPlaceholderText(/new deck name/i)
    await fireEvent.input(input, { target: { value: 'Warblers' } })
    await fireEvent.keyDown(input, { key: 'Enter' })
    expect(onCreate).toHaveBeenCalledWith('Warblers')
  })

  it('shows Creating… and disables while onCreate is pending', async () => {
    let resolve!: () => void
    const onCreate = vi.fn().mockReturnValue(new Promise<void>((r) => { resolve = r }))
    render(DeckCreateForm, { label: '+ Create deck', onCreate })
    await fireEvent.input(screen.getByPlaceholderText(/new deck name/i), {
      target: { value: 'Warblers' },
    })
    await fireEvent.click(screen.getByRole('button', { name: /create deck/i }))
    expect(screen.getByRole('button', { name: /creating/i })).toBeDisabled()
    resolve()
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /create deck/i })).toBeInTheDocument()
    })
  })

  it('clears the input when onCreate resolves', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    render(DeckCreateForm, { label: '+ Create deck', onCreate })
    const input = screen.getByPlaceholderText(/new deck name/i) as HTMLInputElement
    await fireEvent.input(input, { target: { value: 'Warblers' } })
    await fireEvent.click(screen.getByRole('button', { name: /create deck/i }))
    await vi.waitFor(() => expect(input.value).toBe(''))
  })

  it('keeps the input when onCreate rejects so the user can retry', async () => {
    const onCreate = vi.fn().mockRejectedValue(new Error('boom'))
    render(DeckCreateForm, { label: '+ Create deck', onCreate })
    const input = screen.getByPlaceholderText(/new deck name/i) as HTMLInputElement
    await fireEvent.input(input, { target: { value: 'Warblers' } })
    await fireEvent.click(screen.getByRole('button', { name: /create deck/i }))
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /create deck/i })).not.toBeDisabled()
    })
    expect(input.value).toBe('Warblers')
  })

  it('calls onEscape when Escape is pressed in the input', async () => {
    const onEscape = vi.fn()
    render(DeckCreateForm, { label: '+ Create deck', onCreate: vi.fn(), onEscape })
    await fireEvent.keyDown(screen.getByPlaceholderText(/new deck name/i), { key: 'Escape' })
    expect(onEscape).toHaveBeenCalledOnce()
  })
})

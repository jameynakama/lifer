import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import SpeciesTypeahead from './SpeciesTypeahead.svelte'
import type { Species } from '../types'

const species: Species[] = [
  { id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
  { id: 3, common_name: 'Dark-eyed Junco', scientific_name: 'Junco hyemalis', ebird_code: 'daejun' },
]

describe('SpeciesTypeahead', () => {
  it('renders a text input', () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('does not show dropdown for input shorter than 2 chars', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 's' } })
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('shows filtered results for input of 2+ chars', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'sp' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Fox Sparrow')).toBeInTheDocument()
    expect(screen.queryByText('Dark-eyed Junco')).toBeNull()
  })

  it('matches on scientific name', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'mel' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
  })

  it('limits results to 10', async () => {
    const manySpecies: Species[] = Array.from({ length: 15 }, (_, i) => ({
      id: i + 1,
      common_name: `Sparrow ${i + 1}`,
      scientific_name: `Species ${i + 1}`,
      ebird_code: `sp${i + 1}`,
    }))
    render(SpeciesTypeahead, { props: { species: manySpecies, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'sp' } })
    const items = screen.getAllByRole('option')
    expect(items.length).toBe(10)
  })

  it('calls onSelect with the species when a result is clicked', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'song' } })
    const item = screen.getByText('Song Sparrow').closest('[role="option"]')
    await fireEvent.mouseDown(item!)
    expect(onSelect).toHaveBeenCalledWith(species[0])
  })

  it('fills the input with the species common name after selection', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'song' } })
    const item = screen.getByText('Song Sparrow').closest('[role="option"]')
    await fireEvent.mouseDown(item!)
    expect(screen.getByRole('combobox')).toHaveValue('Song Sparrow')
  })

  it('closes the dropdown after selection', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'song' } })
    const item = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(item)
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('calls onSelect(null) when input is cleared below 2 chars', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'so' } })
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 's' } })
    expect(onSelect).toHaveBeenLastCalledWith(null)
  })

  it('selects the first highlighted item on Enter', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    const input = screen.getByRole('combobox')
    await fireEvent.input(input, { target: { value: 'sp' } })
    await fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ common_name: 'Song Sparrow' }))
  })

  it('closes the dropdown on Escape without selecting', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    const input = screen.getByRole('combobox')
    await fireEvent.input(input, { target: { value: 'sp' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    await fireEvent.keyDown(input, { key: 'Escape' })
    expect(screen.queryByRole('listbox')).toBeNull()
    expect(onSelect).not.toHaveBeenCalled()
  })
})

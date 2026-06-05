import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import SpeciesSearchList from './SpeciesSearchList.svelte'

afterEach(() => {
  vi.restoreAllMocks()
})

const sparrow = {
  ebird_code: 'sonspa',
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  image_url: null,
}
const robin = {
  ebird_code: 'amro',
  common_name: 'American Robin',
  scientific_name: 'Turdus migratorius',
  image_url: null,
}
const crow = {
  ebird_code: 'amcro',
  common_name: 'American Crow',
  scientific_name: 'Corvus brachyrhynchos',
  image_url: null,
}

function make(overrides = {}) {
  return {
    results: [sparrow, robin],
    pinned: [],
    addedCodes: new Set<string>(),
    selectedCodes: new Set<string>(),
    onAdd: vi.fn(),
    onToggle: vi.fn(),
    ...overrides,
  }
}

describe('SpeciesSearchList', () => {
  it('renders results', () => {
    render(SpeciesSearchList, make())
    expect(screen.getByText(/song sparrow/i)).toBeInTheDocument()
    expect(screen.getByText(/american robin/i)).toBeInTheDocument()
  })

  it('shows a checkbox per non-added result', () => {
    render(SpeciesSearchList, make())
    expect(screen.getAllByRole('checkbox')).toHaveLength(2)
  })

  it('shows Added indicator instead of checkbox for already-added species', () => {
    render(SpeciesSearchList, make({ addedCodes: new Set(['sonspa']) }))
    expect(screen.getAllByRole('checkbox')).toHaveLength(1)
    expect(screen.getByText(/added/i)).toBeInTheDocument()
    // only sparrow is added; robin still shows Add button
    expect(screen.getAllByRole('button', { name: /^add$/i })).toHaveLength(1)
  })

  it('calls onToggle when checkbox clicked', async () => {
    const onToggle = vi.fn()
    render(SpeciesSearchList, make({ onToggle }))
    const [first] = screen.getAllByRole('checkbox')
    await fireEvent.click(first)
    expect(onToggle).toHaveBeenCalledWith('sonspa')
  })

  it('calls onAdd when Add button clicked', async () => {
    const onAdd = vi.fn()
    render(SpeciesSearchList, make({ onAdd }))
    const [firstAdd] = screen.getAllByRole('button', { name: /^add$/i })
    await fireEvent.click(firstAdd)
    expect(onAdd).toHaveBeenCalledWith('sonspa')
  })

  it('renders pinned species above results', () => {
    render(SpeciesSearchList, make({ pinned: [crow], results: [sparrow] }))
    const items = screen.getAllByRole('listitem')
    expect(items[0].textContent).toContain('American Crow')
    expect(items[1].textContent).toContain('Song Sparrow')
  })

  it('pinned species have checked checkbox', () => {
    render(SpeciesSearchList, make({ pinned: [crow], results: [sparrow] }))
    const checkboxes = screen.getAllByRole('checkbox')
    expect((checkboxes[0] as HTMLInputElement).checked).toBe(true)
    expect((checkboxes[1] as HTMLInputElement).checked).toBe(false)
  })

  it('shows select-all button with count when onSelectAll provided', () => {
    render(SpeciesSearchList, make({ onSelectAll: vi.fn() }))
    expect(screen.getByRole('button', { name: /select all \(2\)/i })).toBeInTheDocument()
  })

  it('does not show select-all button when onSelectAll is not provided', () => {
    render(SpeciesSearchList, make())
    expect(screen.queryByRole('button', { name: /select all/i })).toBeNull()
  })

  it('calls onSelectAll with all non-added result codes when clicked', async () => {
    const onSelectAll = vi.fn()
    render(SpeciesSearchList, make({ onSelectAll, addedCodes: new Set(['sonspa']) }))
    await fireEvent.click(screen.getByRole('button', { name: /select all/i }))
    expect(onSelectAll).toHaveBeenCalledWith(['amro'])
  })

  it('renders nothing when results and pinned are empty', () => {
    render(SpeciesSearchList, make({ results: [], pinned: [] }))
    expect(screen.queryByRole('list')).toBeNull()
  })
})

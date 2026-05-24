<script lang="ts">
  import type { Species } from '../types'

  let { species, onSelect }: {
    species: Species[]
    onSelect: (s: Species | null) => void
  } = $props()

  let query = $state('')
  let highlighted = $state(0)
  let open = $state(false)

  const filtered = $derived(
    query.length < 2
      ? []
      : species
          .filter((s) => {
            const q = query.toLowerCase()
            return (
              s.common_name.toLowerCase().includes(q) ||
              s.scientific_name.toLowerCase().includes(q)
            )
          })
          .slice(0, 10)
  )

  function handleInput() {
    highlighted = 0
    open = query.length >= 2
    if (query.length < 2) {
      onSelect(null)
    }
  }

  function selectSpecies(s: Species) {
    query = s.common_name
    open = false
    highlighted = 0
    onSelect(s)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      highlighted = Math.min(highlighted + 1, filtered.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      highlighted = Math.max(highlighted - 1, 0)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (open && filtered[highlighted]) selectSpecies(filtered[highlighted])
    } else if (e.key === 'Escape') {
      open = false
    }
  }
</script>

<div class="typeahead">
  <input
    type="text"
    bind:value={query}
    oninput={handleInput}
    onkeydown={handleKeydown}
    placeholder="Type species name..."
    class="typeahead-input"
    autocomplete="off"
    role="combobox"
    aria-expanded={open}
    aria-controls="typeahead-dropdown"
    aria-autocomplete="list"
    aria-activedescendant={open && filtered.length > 0 ? 'typeahead-option-' + highlighted : undefined}
  />
  {#if open && filtered.length > 0}
    <ul class="dropdown" role="listbox" id="typeahead-dropdown">
      {#each filtered as s, i (s.id)}
        <li
          class="dropdown-item"
          class:highlighted={i === highlighted}
          role="option"
          aria-selected={i === highlighted}
          id="typeahead-option-{i}"
          onmousedown={() => selectSpecies(s)}
        >
          <span class="common">{s.common_name}</span>
          <span class="scientific">{s.scientific_name}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .typeahead {
    position: relative;
  }
  .typeahead-input {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.6875rem 0.875rem;
    width: 100%;
    font-size: 0.875rem;
    color: var(--text);
    font-family: inherit;
    outline: none;
    box-sizing: border-box;
  }
  .typeahead-input:focus {
    border-color: var(--accent);
  }
  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    list-style: none;
    margin: 0;
    padding: 0.25rem 0;
    z-index: 10;
    box-shadow: var(--shadow);
    max-height: 280px;
    overflow-y: auto;
  }
  .dropdown-item {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    padding: 0.5rem 0.875rem;
    cursor: pointer;
  }
  .dropdown-item.highlighted,
  .dropdown-item:hover {
    background: var(--accent);
  }
  .dropdown-item.highlighted .common,
  .dropdown-item.highlighted .scientific,
  .dropdown-item:hover .common,
  .dropdown-item:hover .scientific {
    color: #fff;
  }
  .common {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text);
  }
  .scientific {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-style: italic;
  }
</style>

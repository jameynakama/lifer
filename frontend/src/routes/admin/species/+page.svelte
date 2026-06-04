<script lang="ts">
  import { page } from '$app/state'

  interface SpeciesResult {
    ebird_code: string
    common_name: string
    scientific_name: string
    image_url: string | null
  }

  interface SpeciesPage {
    results: SpeciesResult[]
    count: number
    next: string | null
    previous: string | null
  }

  let results: SpeciesResult[] = $state([])
  let count = $state(0)
  let next: string | null = $state(null)
  let previous: string | null = $state(null)
  let loading = $state(false)
  let error = $state('')

  const q = $derived(page.url.searchParams.get('q') ?? '')
  const offset = $derived(Number(page.url.searchParams.get('offset') ?? '0'))
  const limit = 25

  $effect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (q) params.set('q', q)
    loading = true
    error = ''
    fetch(`/api/v1/species?${params}`)
      .then((r) => {
        if (!r.ok) throw new Error(`Failed to load: ${r.status}`)
        return r.json()
      })
      .then((data: SpeciesPage) => {
        results = data.results
        count = data.count
        next = data.next
        previous = data.previous
      })
      .catch((e: Error) => {
        error = e.message
      })
      .finally(() => {
        loading = false
      })
  })
</script>

<form method="GET">
  <input type="text" name="q" value={q} placeholder="Search species by name or code" />
  <button type="submit">Search</button>
</form>

<p>{count} species{q ? ` matching "${q}"` : ''}</p>

{#if loading}
  <p>Loading...</p>
{:else if error}
  <p class="error">{error}</p>
{:else}
  <ul>
    {#each results as sp}
      <li>
        <a href="/admin/species/{sp.ebird_code}">
          {sp.common_name} <span class="muted">({sp.ebird_code})</span>
        </a>
      </li>
    {/each}
  </ul>
{/if}

<div class="pagination">
  {#if previous}
    <a href="?q={q}&offset={Math.max(0, offset - limit)}">← Previous</a>
  {/if}
  {#if next}
    <a href="?q={q}&offset={offset + limit}">Next →</a>
  {/if}
</div>

<style>
  form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }
  input {
    flex: 1;
    padding: 0.4rem 0.6rem;
  }
  ul {
    list-style: none;
    padding: 0;
  }
  li {
    padding: 0.4rem 0;
    border-bottom: 1px solid var(--border);
  }
  a {
    color: var(--text);
    text-decoration: none;
  }
  a:hover {
    color: var(--text-secondary);
  }
  .muted {
    color: var(--text-muted);
    font-size: 0.875rem;
  }
  .pagination {
    display: flex;
    gap: 1rem;
    margin-top: 1rem;
  }
  .error {
    color: var(--danger);
  }
</style>

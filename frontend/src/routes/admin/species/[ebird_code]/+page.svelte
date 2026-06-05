<script lang="ts">
  import { page } from '$app/state'

  import { ApiError, apiDelete, apiGet, apiPatch } from '$lib/api'
  import type { AdminSpeciesImage, AdminSpeciesRecording } from '../../../../types'

  const ebirdCode = $derived(page.params.ebird_code)

  let images: AdminSpeciesImage[] = $state([])
  let recordings: AdminSpeciesRecording[] = $state([])
  let loading = $state(true)
  let error = $state('')

  async function loadDetail() {
    loading = true
    error = ''
    try {
      const data = await apiGet<{
        images: AdminSpeciesImage[] | null
        recordings: AdminSpeciesRecording[] | null
      }>(`/api/v1/admin/species/${ebirdCode}`)
      images = data.images ?? []
      recordings = data.recordings ?? []
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load species detail.'
    } finally {
      loading = false
    }
  }

  $effect(() => {
    loadDetail()
  })

  async function deleteImage(macaulayID: string) {
    if (!confirm(`Delete image ${macaulayID}?`)) return
    try {
      await apiDelete(`/api/v1/admin/species/${ebirdCode}/images/${macaulayID}`)
      images = images.filter(i => i.macaulay_id !== macaulayID)
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        alert('This image is locked and cannot be deleted.')
      } else {
        alert('Delete failed')
      }
    }
  }

  async function toggleImageLocked(macaulayID: string, locked: boolean) {
    try {
      await apiPatch(`/api/v1/admin/species/${ebirdCode}/images/${macaulayID}/locked`, { locked })
      images = images.map(i => i.macaulay_id === macaulayID ? { ...i, locked } : i)
    } catch {
      // leave toggle unchanged
    }
  }

  async function deleteRecording(xenoCantoID: string) {
    if (!confirm(`Delete recording ${xenoCantoID}?`)) return
    try {
      await apiDelete(`/api/v1/admin/species/${ebirdCode}/recordings/${xenoCantoID}`)
      recordings = recordings.filter(r => r.xeno_canto_id !== xenoCantoID)
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        alert('This recording is locked and cannot be deleted.')
      } else {
        alert('Delete failed')
      }
    }
  }

  async function toggleRecordingLocked(xenoCantoID: string, locked: boolean) {
    try {
      await apiPatch(`/api/v1/admin/species/${ebirdCode}/recordings/${xenoCantoID}/locked`, { locked })
      recordings = recordings.map(r => r.xeno_canto_id === xenoCantoID ? { ...r, locked } : r)
    } catch {
      // leave toggle unchanged
    }
  }

  // Uploads stay on raw fetch: FormData must not go through the JSON client.
  async function uploadImage(e: SubmitEvent) {
    e.preventDefault()
    const form = e.target as HTMLFormElement
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/images`, {
      method: 'POST',
      body: new FormData(form),
    })
    if (res.ok) {
      const img: AdminSpeciesImage = await res.json()
      images = [...images, img]
      form.reset()
    } else {
      alert('Upload failed')
    }
  }

  async function uploadRecording(e: SubmitEvent) {
    e.preventDefault()
    const form = e.target as HTMLFormElement
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/recordings`, {
      method: 'POST',
      body: new FormData(form),
    })
    if (res.ok) {
      const rec: AdminSpeciesRecording = await res.json()
      recordings = [...recordings, rec]
      form.reset()
    } else {
      alert('Upload failed')
    }
  }
</script>

<a href="/admin/species">← Back to search</a>
<h2>{ebirdCode}</h2>

{#if loading}
  <p>Loading...</p>
{:else if error}
  <p class="error">{error}</p>
{:else}
  <section>
    <h3>Images ({images.length})</h3>
    <div class="image-grid">
      {#each images as img}
        <div class="image-card" class:locked={img.locked}>
          <img src={img.file_path} alt={img.credit} />
          <p class="credit">{img.credit}</p>
          <p class="id">{img.macaulay_id}</p>
          <div class="card-actions">
            <button
              class="btn-lock"
              class:active={img.locked}
              onclick={() => toggleImageLocked(img.macaulay_id, !img.locked)}
            >{img.locked ? '🔒 Locked' : '🔓 Lock'}</button>
            <button
              class="btn-delete btn-danger-ghost"
              disabled={img.locked}
              onclick={() => deleteImage(img.macaulay_id)}
            >Delete</button>
          </div>
        </div>
      {/each}
    </div>

    <details>
      <summary>Upload new image</summary>
      <form onsubmit={uploadImage} enctype="multipart/form-data">
        <label>File: <input type="file" name="file" accept="image/*" required /></label>
        <label>Credit: <input type="text" name="credit" placeholder="Photographer name" /></label>
        <button type="submit">Upload</button>
      </form>
    </details>
  </section>

  <section>
    <h3>Recordings ({recordings.length})</h3>
    <table>
      <thead>
        <tr><th>ID</th><th>Quality</th><th>Type</th><th>Credit</th><th>Actions</th></tr>
      </thead>
      <tbody>
        {#each recordings as rec}
          <tr class:locked={rec.locked}>
            <td>{rec.xeno_canto_id}</td>
            <td>{rec.quality}</td>
            <td>{rec.type}</td>
            <td>{rec.credit}</td>
            <td class="actions-cell">
              <audio src={rec.file_path} controls></audio>
              <button
                class="btn-lock"
                class:active={rec.locked}
                onclick={() => toggleRecordingLocked(rec.xeno_canto_id, !rec.locked)}
              >{rec.locked ? '🔒' : '🔓'}</button>
              <button
                class="btn-delete btn-danger-ghost"
                disabled={rec.locked}
                onclick={() => deleteRecording(rec.xeno_canto_id)}
              >Delete</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>

    <details>
      <summary>Upload new recording</summary>
      <form onsubmit={uploadRecording} enctype="multipart/form-data">
        <label>File: <input type="file" name="file" accept="audio/*" required /></label>
        <label>
          Quality:
          <select name="quality">
            <option>A</option><option>B</option><option>C</option><option>D</option><option>E</option>
          </select>
        </label>
        <label>
          Type:
          <select name="type">
            <option>song</option><option>call</option>
          </select>
        </label>
        <label>
            Credit: <input type="text" name="credit" placeholder="Recorder name" />
        </label>
        <button type="submit">Upload</button>
      </form>
    </details>
  </section>
{/if}

<style>
  section { margin-top: 2rem; }
  .image-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 1rem; margin: 1rem 0; }
  .image-card { border: 1px solid var(--border); border-radius: 6px; padding: 0.5rem; }
  .image-card img { width: 100%; aspect-ratio: 1; object-fit: cover; border-radius: 4px; }
  .image-card.locked { border-color: color-mix(in srgb, var(--accent) 40%, var(--border)); }
  .credit, .id { font-size: 0.75rem; color: var(--text-muted); margin: 0.25rem 0; }
  .card-actions { display: flex; gap: 0.25rem; margin-top: 0.25rem; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid var(--border); font-size: 0.875rem; }
  tr.locked td { background: color-mix(in srgb, var(--accent) 5%, transparent); }
  .actions-cell { display: flex; align-items: center; gap: 0.375rem; flex-wrap: wrap; }
  audio { height: 28px; vertical-align: middle; }
  .btn-lock {
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0.2rem 0.4rem;
    font-size: 0.75rem;
    cursor: pointer;
    font-family: inherit;
    color: var(--text-muted);
  }
  .btn-lock.active {
    border-color: var(--accent);
    color: var(--accent);
  }
  .btn-delete {
    padding: 0.2rem 0.4rem;
    font-size: 0.75rem;
  }
  .btn-delete:disabled { opacity: 0.4; cursor: not-allowed; }
  details { margin-top: 1rem; }
  summary { cursor: pointer; color: var(--text-secondary); }
  form { display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.75rem; max-width: 400px; }
  label { display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem; }
  .error { color: var(--danger); }
</style>

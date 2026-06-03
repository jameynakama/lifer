<script lang="ts">
  import { page } from '$app/state'

  interface SpeciesImage {
    macaulay_id: string
    file_path: string
    credit: string
  }

  interface SpeciesRecording {
    xeno_canto_id: string
    file_path: string
    quality: string
    type: string
  }

  const ebirdCode = $derived(page.params.ebird_code)

  let images: SpeciesImage[] = $state([])
  let recordings: SpeciesRecording[] = $state([])
  let loading = $state(true)
  let error = $state('')

  async function loadDetail() {
    loading = true
    error = ''
    try {
      const res = await fetch(`/api/v1/admin/species/${ebirdCode}`)
      if (!res.ok) throw new Error(`Failed to load: ${res.status}`)
      const data = await res.json()
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
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/images/${macaulayID}`, { method: 'DELETE' })
    if (res.ok) {
      images = images.filter(i => i.macaulay_id !== macaulayID)
    } else {
      alert('Delete failed')
    }
  }

  async function deleteRecording(xenoCantoID: string) {
    if (!confirm(`Delete recording ${xenoCantoID}?`)) return
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/recordings/${xenoCantoID}`, { method: 'DELETE' })
    if (res.ok) {
      recordings = recordings.filter(r => r.xeno_canto_id !== xenoCantoID)
    } else {
      alert('Delete failed')
    }
  }

  async function uploadImage(e: SubmitEvent) {
    e.preventDefault()
    const form = e.target as HTMLFormElement
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/images`, {
      method: 'POST',
      body: new FormData(form),
    })
    if (res.ok) {
      const img: SpeciesImage = await res.json()
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
      const rec: SpeciesRecording = await res.json()
      recordings = [...recordings, rec]
      form.reset()
    } else {
      alert('Upload failed')
    }
  }
</script>

<a href="/admin">← Back to search</a>
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
        <div class="image-card">
          <img src={img.file_path} alt={img.credit} />
          <p class="credit">{img.credit}</p>
          <p class="id">{img.macaulay_id}</p>
          <button onclick={() => deleteImage(img.macaulay_id)}>Delete</button>
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
        <tr><th>ID</th><th>Quality</th><th>Type</th><th>Actions</th></tr>
      </thead>
      <tbody>
        {#each recordings as rec}
          <tr>
            <td>{rec.xeno_canto_id}</td>
            <td>{rec.quality}</td>
            <td>{rec.type}</td>
            <td>
              <audio src={rec.file_path} controls></audio>
              <button onclick={() => deleteRecording(rec.xeno_canto_id)}>Delete</button>
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
  .credit, .id { font-size: 0.75rem; color: var(--text-muted); margin: 0.25rem 0; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid var(--border); font-size: 0.875rem; }
  audio { height: 28px; vertical-align: middle; }
  details { margin-top: 1rem; }
  summary { cursor: pointer; color: var(--text-secondary); }
  form { display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.75rem; max-width: 400px; }
  label { display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem; }
  .error { color: var(--danger, red); }
</style>

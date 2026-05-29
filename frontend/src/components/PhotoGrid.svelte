<script lang="ts">
  let {
    images,
  }: {
    images: { macaulay_id: string; file_path: string; credit: string }[]
  } = $props()
</script>

{#if images.length > 0}
  <div class="photo-grid">
    {#each images as img (img.macaulay_id)}
      <a href={img.file_path} target="_blank" rel="noopener noreferrer" aria-label="View full resolution photo{img.credit ? ` by ${img.credit}` : ''}">
        <img
          src={img.file_path}
          alt={img.credit || 'Species photo'}
          title={img.credit}
          loading="lazy"
        />
      </a>
    {/each}
  </div>
{:else}
  <p class="empty">No photos available.</p>
{/if}

<style>
  .photo-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.375rem;
  }

  .photo-grid a {
    display: block;
    border-radius: 6px;
    overflow: hidden;
  }

  .photo-grid img {
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    border-radius: 6px;
    background: var(--border);
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin: 0;
  }
</style>

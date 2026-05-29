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
      <img
        src={img.file_path}
        alt={img.credit || 'Species photo'}
        title={img.credit}
        loading="lazy"
      />
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

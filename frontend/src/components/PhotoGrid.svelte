<script lang="ts">
  import type { SpeciesImage } from '../types'

  let { images }: { images: SpeciesImage[] } = $props()
</script>

{#if images.length > 0}
  <div class="photo-grid">
    {#each images as img (img.macaulay_id)}
      <a href={img.file_path} target="_blank" rel="noopener noreferrer" aria-label="View full resolution photo{img.credit ? ` by ${img.credit}` : ''}">
        <img
          src={img.file_path}
          alt={img.credit || 'Species photo'}
          loading="lazy"
        />
        {#if img.credit}
          <span class="credit">{img.credit}</span>
        {/if}
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
    position: relative;
  }

  .photo-grid img {
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    border-radius: 6px;
    background: var(--border);
    display: block;
  }

  .credit {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    padding: 0.25rem 0.375rem;
    background: linear-gradient(transparent, rgba(0, 0, 0, 0.65));
    color: rgba(255, 255, 255, 0.9);
    font-size: 0.625rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    opacity: 0;
    transition: opacity 0.15s ease;
  }

  .photo-grid a:hover .credit {
    opacity: 1;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin: 0;
  }
</style>

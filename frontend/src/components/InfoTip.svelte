<script lang="ts">
  let { text }: { text: string } = $props()

  let open = $state(false)
  let root: HTMLElement | undefined = $state()

  // Click-to-toggle (hover doesn't exist on phones). The window handlers
  // close on outside-click and Escape; clicks inside the root (the button
  // itself or the popover) are exempt so toggling works.
  function onWindowClick(e: MouseEvent) {
    if (open && root && !root.contains(e.target as Node)) open = false
  }
  function onWindowKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false
  }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onWindowKeydown} />

<span class="infotip" bind:this={root}>
  <button
    type="button"
    class="infotip-btn"
    aria-label="What does this panel show?"
    aria-expanded={open}
    onclick={() => (open = !open)}
  >
    i
  </button>
  {#if open}
    <span role="tooltip" class="infotip-pop">{text}</span>
  {/if}
</span>

<style>
  .infotip {
    position: relative;
    display: inline-block;
    margin-left: 0.375rem;
  }
  .infotip-btn {
    width: 0.8rem;
    height: 0.8rem;
    padding: 0;
    font-family: inherit;
    font-size: 0.625rem;
    font-style: italic;
    font-weight: 700;
    line-height: 1;
    color: var(--text-muted);
    background: transparent;
    border: 1px solid var(--text-muted);
    border-radius: 50%;
    cursor: pointer;
    vertical-align: baseline;
  }
  .infotip-btn:hover,
  .infotip-btn[aria-expanded='true'] {
    color: var(--accent);
    border-color: var(--accent);
  }
  .infotip-pop {
    position: absolute;
    top: calc(100% + 0.375rem);
    left: 0;
    z-index: 10;
    width: 16rem;
    max-width: 70vw;
    padding: 0.5rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 400;
    line-height: 1.45;
    letter-spacing: normal;
    text-transform: none;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    box-shadow: var(--shadow);
  }
</style>

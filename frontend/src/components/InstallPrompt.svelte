<script lang="ts">
  const isStandalone = window.matchMedia('(display-mode: standalone)').matches
  const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent)
  const isAndroid = /Android/.test(navigator.userAgent)

  let dismissed = $state(!!localStorage.getItem('dismissed-install-prompt'))

  const show = $derived(!dismissed && !isStandalone && (isIOS || isAndroid))

  function dismiss() {
    localStorage.setItem('dismissed-install-prompt', '1')
    dismissed = true
  }
</script>

{#if show}
  <div class="install-prompt">
    <p class="message">
      {#if isIOS}
        Tap the share button then "Add to Home Screen" for quick access.
      {:else}
        Tap ⋮ then "Add to Home Screen" for quick access.
      {/if}
    </p>
    <button class="dismiss" onclick={dismiss} aria-label="Dismiss">✕</button>
  </div>
{/if}

<style>
  .install-prompt {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    background: color-mix(in srgb, var(--accent) 10%, var(--surface));
    border: 1px solid color-mix(in srgb, var(--accent) 30%, transparent);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    box-shadow: var(--shadow);
  }
  .message {
    flex: 1;
    margin: 0;
    font-size: 0.875rem;
    color: var(--text);
  }
  .dismiss {
    flex-shrink: 0;
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 0.875rem;
    cursor: pointer;
    padding: 0.25rem;
    line-height: 1;
  }
</style>

<script lang="ts">
  let { label, onCreate, onEscape }: {
    /** Create-button label, e.g. "+ Create deck". */
    label: string
    /** Creates the deck. The input clears on success and stays on failure for retry. */
    onCreate: (name: string) => void | Promise<void>
    onEscape?: () => void
  } = $props()

  let name = $state('')
  let creating = $state(false)

  async function create() {
    const trimmed = name.trim()
    if (!trimmed) return
    creating = true
    try {
      await onCreate(trimmed)
      name = ''
    } catch {
      // creation failed; input stays for retry
    } finally {
      creating = false
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') create()
    if (e.key === 'Escape') onEscape?.()
  }
</script>

<div class="create-section">
  <input
    class="create-input"
    type="text"
    placeholder="New deck name…"
    bind:value={name}
    onkeydown={handleKeydown}
  />
  <button
    class="create-btn btn-primary"
    onclick={create}
    disabled={!name.trim() || creating}
  >
    {creating ? 'Creating…' : label}
  </button>
</div>

<style>
  .create-section {
    padding: 0.625rem 0.75rem;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }
  .create-input {
    background: var(--bg);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 6px;
    padding: 0.375rem 0.5rem;
    font-size: 0.8125rem;
    font-family: inherit;
    width: 100%;
    box-sizing: border-box;
  }
  .create-btn {
    padding: 0.3125rem 0.625rem;
    font-size: 0.75rem;
    width: 100%;
  }
  .create-btn:disabled {
    opacity: 0.6;
  }
</style>

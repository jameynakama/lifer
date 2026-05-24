<script lang="ts">
  import WaveSurfer from 'wavesurfer.js'
  import { onMount } from 'svelte'

  let { url }: { url: string } = $props()

  let container: HTMLDivElement
  let ws: WaveSurfer
  let playing = $state(false)
  let ready = $state(false)

  onMount(() => {
    const style = getComputedStyle(document.documentElement)
    const waveColor = style.getPropertyValue('--text-secondary').trim() || '#94a3b8'
    const progressColor = style.getPropertyValue('--accent').trim() || '#2563eb'

    ws = WaveSurfer.create({
      container,
      url,
      waveColor,
      progressColor,
      cursorColor: 'transparent',
      height: 60,
      barWidth: 2,
      barGap: 1,
      barRadius: 2,
    })

    ws.on('ready', () => { ready = true })
    ws.on('play', () => { playing = true })
    ws.on('pause', () => { playing = false })
    ws.on('finish', () => { playing = false })

    return () => ws.destroy()
  })

  function togglePlay() {
    ws?.playPause()
  }
</script>

<div class="wave-player">
  <button
    class="play-btn"
    onclick={togglePlay}
    disabled={!ready}
    aria-label={playing ? 'Pause' : 'Play'}
  >
    {#if !ready}
      <span class="spinner" aria-hidden="true"></span>
    {:else if playing}
      ⏸
    {:else}
      ▶
    {/if}
  </button>
  <div class="waveform" bind:this={container}></div>
</div>

<style>
  .wave-player {
    display: flex;
    align-items: center;
    gap: 0.875rem;
    padding: 0.5rem 0;
  }
  .play-btn {
    flex-shrink: 0;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: var(--accent);
    color: #fff;
    border: none;
    font-size: 1.125rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    padding-left: 0.125rem; /* optical nudge for ▶ */
  }
  .play-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .play-btn:not(:disabled):active {
    opacity: 0.85;
  }
  .waveform {
    flex: 1;
    min-width: 0;
  }
  .spinner {
    display: block;
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255, 255, 255, 0.35);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>

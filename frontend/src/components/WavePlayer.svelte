<script lang="ts">
  import WaveSurfer from 'wavesurfer.js'
  import { onMount } from 'svelte'

  let { url, peaks }: { url: string; peaks?: number[] } = $props()

  let container: HTMLDivElement
  let ws: WaveSurfer
  let audio: HTMLAudioElement
  let playing = $state(false)
  let ready = $state(false)

  // WaveSurfer's default mode fetches the audio via XHR to decode waveform
  // peaks. We skip that decode entirely: hand WaveSurfer a native <audio>
  // element and supply peaks ourselves, so the bars draw instantly and playback
  // never waits on a full download.
  //
  // Real peaks are precomputed at ingest and stored per recording. Recordings
  // the transcode backfill has not reached yet have none, and fall back to these
  // cosmetic bars.
  function generatePeaks(count: number): number[][] {
    return [
      Array.from({ length: count }, (_, i) => {
        const x = i / count
        const envelope = Math.pow(Math.sin(x * Math.PI), 0.4) * 0.85
        return Math.random() * envelope + 0.05
      }),
    ]
  }

  // Stored peaks are 0..255 per bucket; WaveSurfer wants one array per channel
  // of amplitudes in 0..1.
  function toWaveSurferPeaks(stored: number[]): number[][] {
    return [stored.map((p) => p / 255)]
  }

  onMount(() => {
    const style = getComputedStyle(document.documentElement)
    const waveColor = style.getPropertyValue('--text-secondary').trim() || '#94a3b8'
    const progressColor = style.getPropertyValue('--accent').trim() || '#2563eb'

    audio = document.createElement('audio')
    audio.src = url
    audio.preload = 'auto'

    ws = WaveSurfer.create({
      container,
      media: audio,
      peaks: peaks && peaks.length > 0 ? toWaveSurferPeaks(peaks) : generatePeaks(200),
      waveColor,
      progressColor,
      cursorColor: 'transparent',
      height: 60,
      barWidth: 2,
      barGap: 1,
      barRadius: 2,
    })

    ws.on('ready', () => {
      ready = true
    })
    ws.on('play', () => {
      playing = true
    })
    ws.on('pause', () => {
      playing = false
    })
    ws.on('finish', () => {
      playing = false
    })

    return () => {
      audio?.pause()
      ws.destroy()
    }
  })

  export function stop() {
    audio?.pause()
    ws?.pause()
  }

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
  }
</style>

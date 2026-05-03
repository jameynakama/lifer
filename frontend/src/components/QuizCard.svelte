<script lang="ts">
    import type { Card } from '../types'

    let { card, onReveal }: { card: Card, onReveal: () => void } = $props()

    let guess: string = $state("");
</script>

<div>
    <audio controls src={card.recording_path}>
        Your browser does not support audio.
    </audio>
    <input bind:value={guess} type="text" placeholder="Type species name" />
    <button onclick={onReveal}>Reveal answer</button>
</div>

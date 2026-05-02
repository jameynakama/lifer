<script lang="ts">
    interface Card {
      recording_path: string
    }

    let { card, onReveal }: { card: Card, onReveal: () => void } = $props();

    let message: string = $state("");
</script>

<div>
    <audio controls src={card.recording_path}>
        Your browser does not support audio.
    </audio>
    <input bind:value={message} type="text" id="guess" name="guess" placeholder="Type species name" />
    <button onclick={onReveal}>Reveal answer</button>
</div>

<script lang="ts">
  import { onMount } from 'svelte'
  import confetti from 'canvas-confetti'

  onMount(() => {
    const feather = confetti.shapeFromText({ text: '🪶', scalar: 2 })
    const bird = confetti.shapeFromText({ text: '🐦', scalar: 2 })
    const shapes = [feather, feather, feather, feather, bird]

    confetti({ particleCount: 28, angle: 60, spread: 55, origin: { x: 0.1, y: 0.65 }, shapes, scalar: 2 })
    setTimeout(() => {
      confetti({ particleCount: 28, angle: 120, spread: 55, origin: { x: 0.9, y: 0.65 }, shapes, scalar: 2 })
    }, 150)
  })
</script>

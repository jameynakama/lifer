<script lang="ts">
  import type { BirdCard } from '../types'
  import { view } from '../stores/view'
  import QuizCard from '../components/QuizCard.svelte'
  import RevealCard from '../components/RevealCard.svelte'
  import StatsBar from '../components/StatsBar.svelte'

  const MOCK_CARDS: BirdCard[] = [
    {
      id: '1',
      recording_path: '/recordings/song-sparrow.mp3',
      common_name: 'Song Sparrow',
      scientific_name: 'Melospiza melodia',
      photo_path: '/photos/song-sparrow.jpg',
    },
    {
      id: '2',
      recording_path: '/recordings/spotted-towhee.mp3',
      common_name: 'Spotted Towhee',
      scientific_name: 'Pipilo maculatus',
      photo_path: '/photos/spotted-towhee.jpg',
    },
  ]

  let index = $state(0)
  let revealed = $state(false)

  const card = $derived(MOCK_CARDS[index])

  const stats = $derived([
    { label: 'Remaining', value: MOCK_CARDS.length - index },
    { label: 'Reviewed', value: index },
    { label: 'Streak', value: 5 },
  ])

  function onReveal() {
    revealed = true
  }

  function onRate(_rating: number) {
    if (index + 1 >= MOCK_CARDS.length) {
      $view = 'dashboard'
    } else {
      index += 1
      revealed = false
    }
  }
</script>

<div class="quiz">
  <StatsBar {stats} />
  {#if revealed}
    <RevealCard {card} {onRate} />
  {:else}
    <QuizCard {card} {onReveal} />
  {/if}
</div>

<style>
  .quiz {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
</style>

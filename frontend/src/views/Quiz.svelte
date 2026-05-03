<script lang="ts">
  import type { Card } from '../types'
  import { view } from '../stores/view'
  import QuizCard from '../components/QuizCard.svelte'
  import RevealCard from '../components/RevealCard.svelte'

  const MOCK_CARDS: Card[] = [
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

<div>
  {#if revealed}
    <RevealCard {card} {onRate} />
  {:else}
    <QuizCard {card} {onReveal} />
  {/if}
</div>

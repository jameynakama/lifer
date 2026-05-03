<script lang="ts">
    import type { BirdCard } from '../types'

    let { card, onRate }: { card: BirdCard, onRate: (rating: number) => void } = $props()

    const ratings = [
      { label: "Again", value: 1 },
      { label: "Hard", value: 2 },
      { label: "Good", value: 3 },
      { label: "Easy", value: 4 },
    ]
</script>

<div>
    <img src={card.photo_path} alt={card.common_name} />
    <p>{card.scientific_name}</p>

    <p>{card.common_name}</p>

    <ul>
        {#each ratings as rating}
            <li>
                <button onclick={() => onRate(rating.value)}>{rating.label}</button>
            </li>
        {/each}
    </ul>
</div>

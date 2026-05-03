<script lang="ts">
    import type { Stat } from '../types'

    let { stats = [] }: { stats: Stat[] } = $props()
</script>

<div>
    <ul>
        {#each stats as stat}
            <li><span>{stat.label}</span>: <span>{stat.value}</span></li>
        {/each}
    </ul>
</div>

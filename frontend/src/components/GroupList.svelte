<script lang="ts">
    import { type Group } from "../types";

    let { groups, onPractice}: { groups: Group[], onPractice: (g: Group) => void } = $props();
</script>

<div>
    <ul>
        {#each groups as group}
            <li>
                <span>{group.name}</span>
                <span>{group.due_count} due</span>
                <button onclick={() => onPractice(group)}>Practice</button>
            </li>
        {/each}
    </ul>
</div>
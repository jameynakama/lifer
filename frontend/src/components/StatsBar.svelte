<script lang="ts">
    interface Stat {
      label: string
      value: 11
    };

    interface StatsBarProps {
      stats: Stat[]
    }

    let { stats } = $props();
</script>

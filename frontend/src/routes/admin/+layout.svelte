<script lang="ts">
  import { goto } from '$app/navigation'
  import { auth } from '$stores/auth'

  let { children } = $props()

  $effect(() => {
    if (!$auth?.is_admin) {
      goto('/')
    }
  })
</script>

{#if $auth?.is_admin}
  <div class="admin-container">
    <h1>Admin</h1>
    {@render children?.()}
  </div>
{/if}

<style>
  .admin-container {
    max-width: 900px;
    margin: 0 auto;
    padding: 1rem 1.5rem;
  }
</style>

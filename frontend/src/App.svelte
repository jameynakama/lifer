<script lang="ts">
  import { auth } from './stores/auth'
  import { view } from './stores/view'
  import Login from './views/Login.svelte'
  import Dashboard from './views/Dashboard.svelte'
  import Quiz from './views/Quiz.svelte'

  $effect(() => {
    fetch('/api/v1/me')
      .then(async (res) => {
        if (res.ok) {
          const user = await res.json()
          $auth = user
          $view = 'dashboard'
        } else {
          $view = 'login'
        }
      })
      .catch(() => {
        $view = 'login'
      })
  })
</script>

{#if $view === 'login'}
  <Login />
{:else if $view === 'dashboard'}
  <Dashboard />
{:else if $view === 'quiz'}
  <Quiz />
{/if}

<script lang="ts">
  import { auth } from './stores/auth'
  import { view } from './stores/view'
  import Login from './views/Login.svelte'
  import Dashboard from './views/Dashboard.svelte'
  import Quiz from './views/Quiz.svelte'

  let checking = $state(true)

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
      .finally(() => {
        checking = false
      })
  })
</script>

{#if checking}
  <div class="loading">
    <span class="spinner"></span>
  </div>
{:else if $view === 'login'}
  <Login />
{:else if $view === 'dashboard'}
  <Dashboard />
{:else if $view === 'quiz'}
  <Quiz />
{/if}

<style>
  .loading {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
  }
  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #334155;
    border-top-color: #94a3b8;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>

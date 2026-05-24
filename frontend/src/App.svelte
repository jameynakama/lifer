<script lang="ts">
  import { auth } from './stores/auth'
  import { view } from './stores/view'
  import { session } from './stores/session'
  import { getCurrentTheme, toggleTheme } from './lib/theme'
  import Login from './views/Login.svelte'
  import Dashboard from './views/Dashboard.svelte'
  import Quiz from './views/Quiz.svelte'

  let checking = $state(true)
  let theme = $state(getCurrentTheme())

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

  function handleToggle() {
    toggleTheme()
    theme = getCurrentTheme()
  }
</script>

{#if checking}
  <div class="loading">
    <span class="spinner"></span>
  </div>
{:else if $view === 'login'}
  <Login />
{:else}
  <header>
    <span class="wordmark">Lifer</span>
    <button
      onclick={handleToggle}
      aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
    >
      {theme === 'dark' ? '☀️' : '🌙'}
    </button>
  </header>
  {#if $view === 'dashboard'}
    <Dashboard />
  {:else if $view === 'quiz'}
    <Quiz groupId={$session.groupId!} lane={$session.lane!} />
  {/if}
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
    border: 3px solid var(--text-muted);
    border-top-color: var(--text-secondary);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 0 1.25rem;
  }
  .wordmark {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--text);
    letter-spacing: -0.02em;
  }
  header button {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 1rem;
    cursor: pointer;
    line-height: 1;
    box-shadow: var(--shadow);
  }
</style>

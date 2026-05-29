<script lang="ts">
  import '../app.css'
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query'
  import { auth } from '$stores/auth'
  import { getCurrentTheme, toggleTheme } from '$lib/theme'
  import Login from '../views/Login.svelte'

  let { children } = $props()
  let checking = $state(true)
  let theme = $state(getCurrentTheme())

  const queryClient = new QueryClient()

  $effect(() => {
    fetch('/api/v1/me')
      .then(async (res) => {
        if (res.ok) {
          $auth = await res.json()
        }
      })
      .catch(() => {})
      .finally(() => { checking = false })
  })

  function handleToggle() {
    toggleTheme()
    theme = getCurrentTheme()
  }
</script>

<QueryClientProvider client={queryClient}>
  {#if checking}
    <div class="loading">
      <span class="spinner"></span>
    </div>
  {:else if !$auth}
    <Login />
  {:else}
    <div class="app-container">
      <header>
        <a href="/" class="wordmark">Lifer</a>
        <nav>
          <a href="/groups">Groups</a>
          <a href="/explore">Explore</a>
        </nav>
        <button
          onclick={handleToggle}
          aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
      </header>
      <main>
        {@render children?.()}
      </main>
    </div>
  {/if}
</QueryClientProvider>

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
    text-decoration: none;
  }
  nav {
    display: flex;
    gap: 1rem;
  }
  nav a {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 0.875rem;
    font-weight: 500;
  }
  nav a:hover {
    color: var(--text);
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
  .app-container {
    max-width: 480px;
    margin: 0 auto;
    padding: 0 1.5rem 2rem;
  }
  main {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
</style>

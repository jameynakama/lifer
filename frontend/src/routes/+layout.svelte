<script lang="ts">
  import '../app.css'
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query'
  import { apiGet, apiPost } from '$lib/api'
  import { auth } from '$stores/auth'
  import { getCurrentTheme, toggleTheme } from '$lib/theme'
  import type { SessionUser } from '../types'
  import Login from '../views/Login.svelte'

  let { children } = $props()
  let checking = $state(true)
  let theme = $state(getCurrentTheme())

  const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 60_000 } } })

  $effect(() => {
    apiGet<SessionUser>('/api/v1/me')
      .then((user) => { $auth = user })
      .catch(() => {}) // not signed in
      .finally(() => { checking = false })
  })

  function handleToggle() {
    toggleTheme()
    theme = getCurrentTheme()
  }

  async function handleLogout() {
    await apiPost('/api/v1/auth/logout').catch(() => {})
    $auth = null
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
        <a href="/" class="wordmark">🐦‍🔥 FlockDeck</a>
        <nav>
          <a href="/decks">Decks</a>
          <a href="/explore">Explore</a>
          {#if $auth?.is_admin}
            <a href="/admin">Admin</a>
          {/if}
        </nav>
        <div class="header-actions">
          <button
            onclick={handleToggle}
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {theme === 'dark' ? '☀️' : '🌙'}
          </button>
          <button onclick={handleLogout} class="logout">Log out</button>
        </div>
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
  .app-container {
    max-width: 900px;
    margin: 0 auto;
    padding: 0 1.5rem 2rem;
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
  .header-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
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
  .logout {
    font-size: 0.75rem !important;
  }
  main {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  /* Mobile: two-row header */
  @media (max-width: 639px) {
    .app-container {
      padding: 0 1rem 2rem;
    }
    header {
      flex-wrap: wrap;
      padding: 0.75rem 0 0;
      gap: 0.25rem 0;
    }
    .wordmark {
      flex: 1;
    }
    nav {
      order: 3;
      width: 100%;
      padding: 0.375rem 0 0.625rem;
      border-top: 1px solid var(--border);
      margin-top: 0.25rem;
    }
  }
</style>

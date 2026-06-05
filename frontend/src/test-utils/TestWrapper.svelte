<script lang="ts">
  import { QueryClientProvider, type QueryClient } from '@tanstack/svelte-query'
  import type { Component } from 'svelte'

  let {
    component,
    props = {},
    client,
  }: {
    component: Component<Record<string, unknown>>
    props?: Record<string, unknown>
    client: QueryClient
  } = $props()

  const Comp = $derived(component)
</script>

<QueryClientProvider {client}>
  <Comp {...props} />
</QueryClientProvider>

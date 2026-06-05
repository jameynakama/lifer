import { render } from '@testing-library/svelte'
import { QueryClient } from '@tanstack/svelte-query'
import type { Component } from 'svelte'
import TestWrapper from './TestWrapper.svelte'

/**
 * Render a component inside a fresh QueryClientProvider. Required for any
 * component using createQuery/createMutation (e.g. the $lib/queries factories).
 */
export function renderWithClient(
  component: Component<never>,
  props: Record<string, unknown> = {}
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(TestWrapper, {
    props: { component: component as Component<Record<string, unknown>>, props, client },
  })
}

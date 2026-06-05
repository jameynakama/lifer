import { render } from '@testing-library/svelte'
import { QueryClient, type createQuery } from '@tanstack/svelte-query'
import type { Component } from 'svelte'
import TestWrapper from './TestWrapper.svelte'

/**
 * Builds the partial query result components actually read when mocking
 * createQuery. The cast lives here once (a full CreateQueryResult has dozens
 * of fields no component touches) instead of `as any` at every mock site.
 */
export function queryResult<T>(r: {
  data?: T
  isPending?: boolean
  isError?: boolean
}): ReturnType<typeof createQuery> {
  return { isPending: false, isError: false, ...r } as unknown as ReturnType<typeof createQuery>
}

/**
 * Render a component inside a fresh QueryClientProvider. Required for any
 * component using createQuery/createMutation (e.g. the $lib/queries factories).
 */
export function renderWithClient(component: Component<never>, props: Record<string, unknown> = {}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(TestWrapper, {
    props: { component: component as Component<Record<string, unknown>>, props, client },
  })
}

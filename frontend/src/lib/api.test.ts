import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiDelete, apiGet, apiPatch, apiPost, apiPut } from './api'

async function captureError(p: Promise<unknown>): Promise<ApiError> {
  try {
    await p
  } catch (e) {
    return e as ApiError
  }
  throw new Error('expected promise to reject')
}

function jsonResponse(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('apiGet', () => {
  it('should return parsed JSON on success', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ decks: [{ id: 1 }] })))

    const data = await apiGet<{ decks: { id: number }[] }>('/api/v1/decks')

    expect(data.decks[0].id).toBe(1)
    expect(fetch).toHaveBeenCalledWith('/api/v1/decks', expect.objectContaining({ method: 'GET' }))
  })

  it('should throw ApiError with status and server text on failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('forbidden', { status: 403 })))

    const err = await captureError(apiGet('/api/v1/decks/1'))

    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(403)
    expect(err.message).toBe('forbidden')
  })

  it('should use the {error} field when the server returns a JSON error body', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ error: 'name is required' }, 400))
    )

    const err = await captureError(apiGet('/api/v1/decks'))

    expect(err).toBeInstanceOf(ApiError)
    expect(err.message).toBe('name is required')
  })

  it('should fall back to a generic message when the error body is empty', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 500 })))

    const err = await captureError(apiGet('/api/v1/decks'))

    expect(err.message).toBe('Request failed (500)')
  })
})

describe('apiPost', () => {
  it('should send a JSON body with Content-Type header', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ id: 5 })))

    const data = await apiPost<{ id: number }>('/api/v1/decks', { name: 'Yard Birds' })

    expect(data.id).toBe(5)
    expect(fetch).toHaveBeenCalledWith('/api/v1/decks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Yard Birds' }),
    })
  })

  it('should omit body and Content-Type when no body is given', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ id: 9 })))

    await apiPost('/api/v1/decks/3/clone')

    expect(fetch).toHaveBeenCalledWith('/api/v1/decks/3/clone', { method: 'POST' })
  })

  it('should return undefined for 204 responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))

    const data = await apiPost('/api/v1/decks/3/rate', { rating: 3 })

    expect(data).toBeUndefined()
  })
})

describe('apiPut / apiPatch / apiDelete', () => {
  it.each([
    ['apiPut', apiPut, 'PUT'],
    ['apiPatch', apiPatch, 'PATCH'],
  ] as const)('%s should send the body with the right method', async (_name, fn, method) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))

    await fn('/api/v1/things/1', { a: 1 })

    expect(fetch).toHaveBeenCalledWith('/api/v1/things/1', {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ a: 1 }),
    })
  })

  it('apiDelete should send DELETE and tolerate empty responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))

    await expect(apiDelete('/api/v1/decks/7')).resolves.toBeUndefined()
    expect(fetch).toHaveBeenCalledWith('/api/v1/decks/7', { method: 'DELETE' })
  })

  it('apiDelete should throw ApiError on failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('not found', { status: 404 })))

    const err = await captureError(apiDelete('/api/v1/decks/7'))

    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(404)
  })
})

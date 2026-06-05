/**
 * Typed fetch helpers for the FlockDeck API. Every call site shares one
 * error shape (ApiError) and one JSON-handling path.
 */

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

/** Extract a useful message from an error response body. */
async function errorMessage(res: Response): Promise<string> {
  const text = await res.text().catch(() => '')
  if (text) {
    // Backend errors are text/plain today and {error: "..."} JSON later --
    // handle both so the migration is invisible to callers.
    try {
      const parsed = JSON.parse(text)
      if (typeof parsed?.error === 'string') return parsed.error
    } catch {
      // not JSON: fall through to raw text
    }
    return text.trim()
  }
  return `Request failed (${res.status})`
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method }
  if (body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(body)
  }

  const res = await fetch(url, init)
  if (!res.ok) {
    throw new ApiError(res.status, await errorMessage(res))
  }
  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

export const apiGet = <T>(url: string): Promise<T> => request<T>('GET', url)
export const apiPost = <T = void>(url: string, body?: unknown): Promise<T> =>
  request<T>('POST', url, body)
export const apiPut = <T = void>(url: string, body?: unknown): Promise<T> =>
  request<T>('PUT', url, body)
export const apiPatch = <T = void>(url: string, body?: unknown): Promise<T> =>
  request<T>('PATCH', url, body)
export const apiDelete = (url: string): Promise<void> => request<void>('DELETE', url)

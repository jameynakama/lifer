type Theme = 'dark' | 'light'

const STORAGE_KEY = 'lifer-theme'

export function getCurrentTheme(): Theme {
  const attr = document.documentElement.getAttribute('data-theme') as Theme | null
  if (attr === 'dark' || attr === 'light') return attr
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function initTheme(): void {
  const saved = localStorage.getItem(STORAGE_KEY) as Theme | null
  if (saved === 'dark' || saved === 'light') {
    document.documentElement.setAttribute('data-theme', saved)
  }
}

export function toggleTheme(): void {
  const next: Theme = getCurrentTheme() === 'dark' ? 'light' : 'dark'
  document.documentElement.setAttribute('data-theme', next)
  localStorage.setItem(STORAGE_KEY, next)
}

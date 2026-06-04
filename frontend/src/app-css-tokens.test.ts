import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const css = readFileSync('src/app.css', 'utf8')

/** Extract the custom-property names defined inside each top-level block. */
function tokenSets(source: string): Map<string, Set<string>> {
  const sets = new Map<string, Set<string>>()
  const blockRe = /(:root[^{]*|@media[^{]*\{\s*:root[^{]*)\{([^}]*)\}/g
  for (const match of source.matchAll(blockRe)) {
    const selector = match[1].trim()
    const tokens = new Set([...match[2].matchAll(/(--[\w-]+)\s*:/g)].map((m) => m[1]))
    sets.set(selector, tokens)
  }
  return sets
}

describe('app.css theme tokens', () => {
  const sets = tokenSets(css)
  const blocks = [...sets.entries()]

  it('should find the dark, light, and OS-preference theme blocks', () => {
    expect(blocks.length).toBeGreaterThanOrEqual(3)
  })

  it('should define the same token set in every theme block', () => {
    const [first, ...rest] = blocks
    for (const [selector, tokens] of rest) {
      expect([...tokens].sort(), `tokens in "${selector}" vs "${first[0]}"`).toEqual(
        [...first[1]].sort()
      )
    }
  })

  it.each(['--danger', '--error', '--success'])('should define %s', (token) => {
    for (const [selector, tokens] of blocks) {
      expect(tokens.has(token), `${token} missing from "${selector}"`).toBe(true)
    }
  })
})

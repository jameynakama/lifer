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
  // Block 0 is the base :root (dark default + theme-independent structural
  // tokens); blocks 1..n are the light-theme overrides.
  const [base, ...overrides] = blocks

  it('should find the dark, light, and OS-preference theme blocks', () => {
    expect(blocks.length).toBeGreaterThanOrEqual(3)
  })

  it('should define the same token set in every light-override block', () => {
    const [first, ...rest] = overrides
    for (const [selector, tokens] of rest) {
      expect([...tokens].sort(), `tokens in "${selector}" vs "${first[0]}"`).toEqual(
        [...first[1]].sort(),
      )
    }
  })

  it('should define every overridden token in the base block too', () => {
    for (const [selector, tokens] of overrides) {
      for (const token of tokens) {
        expect(base[1].has(token), `${token} from "${selector}" missing in base :root`).toBe(true)
      }
    }
  })

  it.each(['--danger', '--error', '--success'])(
    'should define %s in every theme block',
    (token) => {
      for (const [selector, tokens] of blocks) {
        expect(tokens.has(token), `${token} missing from "${selector}"`).toBe(true)
      }
    },
  )

  it.each(['--radius-sm', '--radius-md', '--radius-lg', '--on-accent', '--shadow-stacked'])(
    'should define structural token %s in the base block',
    (token) => {
      expect(base[1].has(token), `${token} missing from base :root`).toBe(true)
    },
  )

  it('should never set --shadow to the literal none', () => {
    // --shadow composes into --shadow-stacked's shadow list, and a box-shadow
    // LIST containing the keyword none is invalid CSS -- the browser drops the
    // whole declaration (this silently killed the deck-stack effect in dark
    // mode). Use a no-op shadow like "0 0 #0000" instead.
    expect(css).not.toMatch(/--shadow:\s*none/)
  })
})

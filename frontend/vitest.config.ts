import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte({ hot: !process.env.VITEST })],
  resolve: {
    conditions: ['browser'],
    alias: {
      $stores: new URL('./src/stores', import.meta.url).pathname,
      $components: new URL('./src/components', import.meta.url).pathname,
      $lib: new URL('./src/lib', import.meta.url).pathname,
      '$app/navigation': new URL('./src/__mocks__/app-navigation.ts', import.meta.url).pathname,
      '$app/state': new URL('./src/__mocks__/app-state.ts', import.meta.url).pathname,
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/test-setup.ts'],
    globals: true,
    coverage: {
      provider: 'v8',
      include: ['src/**'],
      exclude: ['src/**/*.test.ts', 'src/__mocks__/**', 'src/test-utils/**', 'src/test-setup.ts'],
    },
  },
})

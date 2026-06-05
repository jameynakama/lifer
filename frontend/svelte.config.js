import adapter from '@sveltejs/adapter-static'

export default {
  kit: {
    adapter: adapter({ fallback: 'index.html' }),
    alias: {
      $stores: './src/stores',
      $components: './src/components',
      $lib: './src/lib',
    },
  },
}

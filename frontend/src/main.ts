import { mount } from 'svelte'
import App from './App.svelte'
import { initTheme } from './lib/theme'
import './app.css'

initTheme()

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app

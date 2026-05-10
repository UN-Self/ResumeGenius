import { defineConfig } from 'astro/config'
import tailwind from '@astrojs/tailwind'

export default defineConfig({
  integrations: [tailwind()],
  output: 'static',
  vite: {
    server: {
      proxy: {
        '/app': 'http://localhost:3000',
        '/api': 'http://localhost:8080',
      },
    },
  },
})

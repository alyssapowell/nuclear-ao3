import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

// The site is plain files (HTML, JavaScript, CSS) built into dist/. It talks
// to the Go app only through /api, /ws and /health on the same address.
//
// In production, Caddy serves dist/ and forwards those paths to the app. With
// `npm run dev`, Vite does the same job, forwarding them to the app at
// API_URL (default http://localhost:8088, where docker-compose publishes it).
const api = process.env.API_URL || 'http://localhost:8088'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 3001,
    proxy: {
      '/api': api,
      '/health': api,
      '/ws': { target: api, ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    exclude: ['src/__tests__/test-utils.ts'],
  },
})

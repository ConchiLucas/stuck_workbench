import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  resolve: { dedupe: ['react', 'react-dom'] },
  plugins: [react()],
  server: { host: true, port: 19152, proxy: { '^/api/v1/literacy/items/[0-9]+/(glyph\\.png|sense\\.png|speech\\.mp3)$': { target: process.env.VITE_CONTENT_API_PROXY ?? 'http://localhost:19091', rewrite: path => path.replace('/literacy/items/', '/literacy/chars/') }, '/api': process.env.VITE_API_PROXY ?? 'http://localhost:19151' } },
  test: { environment: 'jsdom', setupFiles: './src/test/setup.ts', exclude: ['e2e/**', 'node_modules/**', 'dist/**'] },
})

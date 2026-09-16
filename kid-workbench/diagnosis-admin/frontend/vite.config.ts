/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: { dedupe: ['react', 'react-dom'] },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
  server: {
    port: 19212,
    proxy: { '/api': 'http://localhost:19211', '/healthz': 'http://localhost:19211' },
  },
  build: {
    outDir: '../backend/internal/http/dist',
    emptyOutDir: true,
  },
})

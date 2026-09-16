/// <reference types="vitest/config" />
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  resolve: { dedupe: ['react', 'react-dom'] },
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
  server: {
    port: 19202,
    proxy: { '/api': loadEnv('development', '.', 'VITE_').VITE_API_PROXY || 'http://localhost:19201', '/healthz': loadEnv('development', '.', 'VITE_').VITE_API_PROXY || 'http://localhost:19201' },
  },
  build: {
    outDir: '../backend/internal/http/dist',
    emptyOutDir: true,
  },
})

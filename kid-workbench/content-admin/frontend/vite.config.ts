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
    port: 19092,
    proxy: { '/api': loadEnv('development', '.', 'VITE_').VITE_API_PROXY || 'http://localhost:19091', '/healthz': loadEnv('development', '.', 'VITE_').VITE_API_PROXY || 'http://localhost:19091' },
  },
  build: {
    outDir: '../backend/internal/http/dist',
    emptyOutDir: true,
  },
})

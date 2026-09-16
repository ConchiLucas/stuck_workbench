import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: { dedupe: ['react', 'react-dom'] },
  server: { host: true, port: 19122, proxy: { '/api': process.env.VITE_API_PROXY ?? 'http://localhost:19121' } },
  test: { environment: 'jsdom', setupFiles: './src/test/setup.ts' },
})

import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: { host: true, port: 19162, proxy: { '/api': process.env.VITE_API_PROXY ?? 'http://localhost:19161' } },
  test: { environment: 'jsdom', setupFiles: './src/test/setup.ts' },
})

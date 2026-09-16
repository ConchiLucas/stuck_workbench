import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  resolve: { dedupe: ['react', 'react-dom'] },
  plugins: [react()],
  server: { host: true, port: 19192, proxy: { '/api': process.env.VITE_API_PROXY ?? 'http://localhost:19191' } },
  test: { environment: 'jsdom', setupFiles: './src/test/setup.ts' },
})

import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: { dedupe: ['react', 'react-dom'] },
  server: { host: true, port: 19142, strictPort: true, proxy: { '/api': process.env.VITE_API_PROXY ?? 'http://localhost:19141' } },
  test: { environment: 'jsdom', setupFiles: './src/test/setup.ts', exclude: ['e2e/**', 'node_modules/**', 'dist/**'] },
})

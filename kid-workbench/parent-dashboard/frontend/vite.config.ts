import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const apiProxy = process.env.VITE_API_PROXY ?? 'http://localhost:19081'

export default defineConfig({
  resolve: { dedupe: ['react', 'react-dom'] },
  plugins: [react()],
  server: {
    port: 19083,
    proxy: { '/api': apiProxy },
  },
  build: {
    outDir: '../backend/internal/http/dist',
    emptyOutDir: true,
  },
})

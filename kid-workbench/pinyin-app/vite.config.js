import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig({
    plugins: [react()],
    server: { host: true, port: 19112, proxy: { '/api': process.env.VITE_API_PROXY ?? 'http://localhost:19111' } },
    test: { environment: 'jsdom', setupFiles: './src/test/setup.ts' },
});

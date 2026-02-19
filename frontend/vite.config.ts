import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/health': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/quotes': {
        target: 'http://localhost:8092',
        changeOrigin: true,
      },
      '/positions': {
        target: 'http://localhost:8092',
        changeOrigin: true,
      },
      '/account': {
        target: 'http://localhost:8092',
        changeOrigin: true,
      },
      // Proxy jax-research health to avoid CORS (browser → 5173 → 8091)
      '/research-health': {
        target: 'http://localhost:8091',
        changeOrigin: true,
        rewrite: () => '/health',
      },
    },
  },
  build: {
    chunkSizeWarningLimit: 1000, // Suppress warning for chunks > 500KB
  },
});

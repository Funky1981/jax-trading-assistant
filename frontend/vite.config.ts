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
      '/reports': {
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
      '/ib-health': {
        target: 'http://localhost:8092',
        changeOrigin: true,
        rewrite: () => '/health',
      },
      '/v1/memory': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
      '/agent0': {
        target: 'http://localhost:8093',
        changeOrigin: true,
        rewrite: (requestPath) => requestPath.replace(/^\/agent0/, ''),
      },
      '/research-health': {
        target: 'http://localhost:8091',
        changeOrigin: true,
        rewrite: () => '/health',
      },
    },
  },
  build: {
    chunkSizeWarningLimit: 1000,
  },
});

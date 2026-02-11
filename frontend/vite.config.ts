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
    },
  },
  build: {
    chunkSizeWarningLimit: 1000, // Suppress warning for chunks > 500KB
  },
});

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../../assets/static',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/storage': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/rest': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/realtime': {
        target: 'http://localhost:54321',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})

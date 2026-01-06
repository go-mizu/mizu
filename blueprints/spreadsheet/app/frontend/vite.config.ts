import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

const isDesktop = process.env.DESKTOP === 'true'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: isDesktop ? '../../assets/static/dist-desktop' : '../../assets/static/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom'],
          'mantine': ['@mantine/core', '@mantine/hooks'],
          'grid': ['@glideapps/glide-data-grid'],
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/static': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
    },
  },
})

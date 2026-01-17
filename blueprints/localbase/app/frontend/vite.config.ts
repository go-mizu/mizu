import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@components': resolve(__dirname, './src/components'),
      '@pages': resolve(__dirname, './src/pages'),
      '@api': resolve(__dirname, './src/api'),
      '@stores': resolve(__dirname, './src/stores'),
      '@hooks': resolve(__dirname, './src/hooks'),
      '@types': resolve(__dirname, './src/types'),
    },
  },
  build: {
    outDir: '../../assets/static',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-mantine': ['@mantine/core', '@mantine/hooks', '@mantine/notifications', '@mantine/dropzone'],
          'vendor-monaco': ['@monaco-editor/react'],
        },
      },
    },
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

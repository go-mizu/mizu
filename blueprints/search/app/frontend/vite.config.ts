import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../../assets/static',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          // Core React runtime - loaded on every page
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          // Markdown rendering - only loaded when needed
          'markdown': ['react-markdown', 'remark-gfm', 'rehype-raw'],
          // UI utilities
          'ui': ['zustand', 'lucide-react'],
        },
      },
    },
  },
})

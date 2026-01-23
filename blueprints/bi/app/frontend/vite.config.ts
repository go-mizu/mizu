import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../../assets/static/dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // React core
          if (id.includes('node_modules/react/') ||
              id.includes('node_modules/react-dom/') ||
              id.includes('node_modules/react-router-dom/')) {
            return 'vendor-react'
          }

          // Mantine UI
          if (id.includes('node_modules/@mantine/')) {
            return 'vendor-mantine'
          }

          // Icons
          if (id.includes('node_modules/@tabler/icons-react')) {
            return 'vendor-icons'
          }

          // Charts (recharts)
          if (id.includes('node_modules/recharts') ||
              id.includes('node_modules/d3-') ||
              id.includes('node_modules/victory-')) {
            return 'vendor-charts'
          }

          // DnD and grid
          if (id.includes('node_modules/@hello-pangea/dnd') ||
              id.includes('node_modules/react-grid-layout')) {
            return 'vendor-dnd'
          }

          // CodeMirror (SQL editor) - this is heavy
          if (id.includes('node_modules/@codemirror/') ||
              id.includes('node_modules/@uiw/react-codemirror') ||
              id.includes('node_modules/@lezer/') ||
              id.includes('node_modules/crelt') ||
              id.includes('node_modules/style-mod') ||
              id.includes('node_modules/w3c-keyname')) {
            return 'vendor-codemirror'
          }

          // SQL formatter
          if (id.includes('node_modules/sql-formatter')) {
            return 'vendor-sql'
          }

          // Utils
          if (id.includes('node_modules/@tanstack/react-query') ||
              id.includes('node_modules/zustand') ||
              id.includes('node_modules/dayjs')) {
            return 'vendor-utils'
          }
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    exclude: ['**/node_modules/**', '**/e2e/**'],
  },
})

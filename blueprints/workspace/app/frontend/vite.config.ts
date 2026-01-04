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
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'src/main.tsx'),
      },
      output: {
        entryFileNames: 'js/[name].js',
        chunkFileNames: 'js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'css/[name][extname]'
          }
          return 'assets/[name]-[hash][extname]'
        },
        // Split large dependencies into separate chunks for better caching
        manualChunks: {
          // React core
          'react-vendor': ['react', 'react-dom'],
          // BlockNote editor (large)
          'blocknote': [
            '@blocknote/core',
            '@blocknote/react',
            '@blocknote/mantine',
            '@blocknote/xl-multi-column',
          ],
          // Syntax highlighting (large - loads many language grammars)
          'shiki': ['shiki'],
          // Data grid for database views
          'datagrid': [
            '@glideapps/glide-data-grid',
            '@glideapps/glide-data-grid-cells',
          ],
          // Emoji picker
          'emoji': ['emoji-mart', '@emoji-mart/react', '@emoji-mart/data'],
          // Math equations
          'katex': ['katex'],
          // Animation library
          'motion': ['framer-motion'],
          // UI components
          'mantine': ['@mantine/core', '@mantine/hooks'],
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        // Don't throw on connection errors in dev mode
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            console.warn('[vite proxy] Backend not available:', err.message)
            // Return a mock response for dev mode
            if (res && !res.headersSent) {
              res.writeHead(503, { 'Content-Type': 'application/json' })
              res.end(JSON.stringify({
                error: 'Backend not available',
                dev_mode: true,
                message: 'Start the backend server or use mock data'
              }))
            }
          })
        },
      },
      '/w': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            console.warn('[vite proxy] Backend not available:', err.message)
            if (res && !res.headersSent) {
              res.writeHead(503, { 'Content-Type': 'application/json' })
              res.end(JSON.stringify({ error: 'Backend not available', dev_mode: true }))
            }
          })
        },
      },
      '/static': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            console.warn('[vite proxy] Backend not available:', err.message)
            if (res && !res.headersSent) {
              res.writeHead(503, { 'Content-Type': 'text/plain' })
              res.end('Backend not available')
            }
          })
        },
      },
    },
  },
})

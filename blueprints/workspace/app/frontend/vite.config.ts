import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

const isDesktop = process.env.DESKTOP === 'true'

export default defineConfig({
  plugins: [react()],
  // Set base path for asset URLs - desktop uses root, web uses /static/dist/
  base: isDesktop ? '/' : '/static/dist/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: isDesktop ? '../desktop/build/frontend' : '../../assets/static/dist',
    emptyOutDir: true,
    // Enable minification and tree-shaking optimizations
    minify: 'esbuild',
    target: 'es2020',
    // Increase chunk size warning limit (we're handling chunking manually)
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      // For desktop builds, use index.html as entry; for web builds, use main.tsx directly
      input: isDesktop
        ? path.resolve(__dirname, 'index.html')
        : { main: path.resolve(__dirname, 'src/main.tsx') },
      output: {
        entryFileNames: 'js/[name].js',
        chunkFileNames: 'js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'css/[name][extname]'
          }
          return 'assets/[name]-[hash][extname]'
        },
        // Optimized chunk splitting strategy for better caching and smaller initial load
        manualChunks(id) {
          // React core - always loaded
          if (id.includes('node_modules/react-dom') || id.includes('node_modules/react/')) {
            return 'react-vendor'
          }

          // Mantine UI - core UI components
          if (id.includes('node_modules/@mantine/')) {
            return 'mantine'
          }

          // BlockNote editor - split into separate chunk (large, lazy-loaded)
          if (id.includes('node_modules/@blocknote/') || id.includes('node_modules/@tiptap/')) {
            return 'blocknote'
          }

          // Shiki syntax highlighting - languages loaded on-demand
          // Only bundle shiki core, grammars are loaded dynamically
          if (id.includes('node_modules/shiki/')) {
            // Split shiki core from language grammars
            if (id.includes('/langs/')) {
              // Each language grammar gets its own chunk for on-demand loading
              const match = id.match(/\/langs\/([^/]+)/)
              if (match) {
                return `shiki-lang-${match[1].replace('.mjs', '')}`
              }
            }
            return 'shiki-core'
          }

          // Data grid - split for database views
          if (id.includes('node_modules/@glideapps/')) {
            return 'datagrid'
          }

          // Emoji picker - lazy loaded
          if (id.includes('node_modules/emoji-mart') || id.includes('node_modules/@emoji-mart/')) {
            return 'emoji'
          }

          // KaTeX math - lazy loaded when equations are used
          if (id.includes('node_modules/katex')) {
            return 'katex'
          }

          // Framer Motion - animations
          if (id.includes('node_modules/framer-motion')) {
            return 'motion'
          }

          // Recharts - charts for database views
          if (id.includes('node_modules/recharts') || id.includes('node_modules/d3-')) {
            return 'charts'
          }

          // PDF viewer - lazy loaded
          if (id.includes('node_modules/react-pdf') || id.includes('node_modules/pdfjs-dist')) {
            return 'pdf'
          }

          // Date utilities
          if (id.includes('node_modules/date-fns') || id.includes('node_modules/dayjs')) {
            return 'date-utils'
          }

          // Lucide icons - tree-shake by keeping in main bundle
          // Individual icons are tree-shaken by rollup
        },
      },
      // Ensure proper tree-shaking
      treeshake: {
        moduleSideEffects: 'no-external',
        propertyReadSideEffects: false,
      },
    },
  },
  // Optimize dependencies for faster dev server and better tree-shaking
  optimizeDeps: {
    include: ['react', 'react-dom', '@mantine/core', '@mantine/hooks'],
    exclude: ['@emoji-mart/data'], // Exclude large data files from pre-bundling
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

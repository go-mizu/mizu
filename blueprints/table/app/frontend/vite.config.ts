import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig(({ mode }) => {
  const isDev = mode === 'development';

  return {
    plugins: [react()],
    // In dev mode, use root path; in production, use static/dist for embedded assets
    base: isDev ? '/' : '/static/dist/',
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    // Optimize dependency pre-bundling for faster dev startup
    optimizeDeps: {
      include: ['react', 'react-dom', 'zustand', 'date-fns'],
    },
    build: {
      outDir: '../../assets/static/dist',
      emptyOutDir: true,
      sourcemap: true,
      rollupOptions: {
        output: {
          entryFileNames: 'js/main.js',
          chunkFileNames: 'js/[name].js',
          assetFileNames: (assetInfo) => {
            if (assetInfo.name?.endsWith('.css')) {
              return 'css/main.css';
            }
            return 'assets/[name][extname]';
          },
          manualChunks(id) {
            // React core - changes rarely
            if (id.includes('node_modules/react-dom') ||
                id.includes('node_modules/react/')) {
              return 'react-vendor';
            }
            // DnD kit - used for drag-drop
            if (id.includes('node_modules/@dnd-kit/')) {
              return 'dnd';
            }
            // Date utilities
            if (id.includes('node_modules/date-fns')) {
              return 'date-fns';
            }
            // State management
            if (id.includes('node_modules/zustand')) {
              return 'zustand';
            }
          },
        },
      },
    },
    server: {
      port: 5173,
      // Enable HMR with better error overlay
      hmr: {
        overlay: true,
      },
      proxy: {
        '/api': {
          target: 'http://localhost:3000',
          changeOrigin: true,
          configure: (proxy) => {
            proxy.on('error', (err, _req, res) => {
              console.warn('\n\x1b[33m[vite proxy]\x1b[0m Backend not available:', err.message);
              console.warn('\x1b[36m  → Start backend with: make backend-dev\x1b[0m\n');
              if (res && !res.headersSent) {
                res.writeHead(503, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                  error: 'Backend not available',
                  dev_mode: true,
                  message: 'Start the backend server with: make backend-dev',
                  hint: 'Run "make backend-dev" in another terminal'
                }));
              }
            });
            proxy.on('proxyReq', (_proxyReq, req) => {
              if (isDev) {
                console.log(`\x1b[90m[proxy]\x1b[0m ${req.method} ${req.url}`);
              }
            });
          },
        },
        '/static': {
          target: 'http://localhost:3000',
          changeOrigin: true,
        },
        '/uploads': {
          target: 'http://localhost:3000',
          changeOrigin: true,
        },
      },
    },
    // Define environment variables for dev mode detection
    define: {
      __DEV__: isDev,
    },
  };
});

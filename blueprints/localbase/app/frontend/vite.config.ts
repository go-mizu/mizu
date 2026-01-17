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
          // Core React runtime
          'vendor-react': ['react', 'react-dom', 'react-router-dom', 'scheduler'],
          // Mantine UI framework
          'vendor-mantine': [
            '@mantine/core',
            '@mantine/hooks',
            '@mantine/notifications',
            '@mantine/dropzone',
            '@mantine/dates',
            '@floating-ui/react',
            '@floating-ui/dom',
            '@floating-ui/core',
          ],
          // Monaco editor loader
          'vendor-monaco': ['@monaco-editor/react'],
          // Icons
          'vendor-icons': ['@tabler/icons-react'],
          // Charts library
          'vendor-charts': ['recharts'],
          // Flow diagrams
          'vendor-xyflow': ['@xyflow/react', 'dagre'],
          // Animation
          'vendor-motion': ['framer-motion'],
          // State management
          'vendor-state': ['zustand'],
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

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig(({ mode }) => ({
  plugins: [react()],
  base: mode === 'development' ? '/' : '/static/dist/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../../assets/static/dist',
    emptyOutDir: true,
    minify: 'esbuild',
    target: 'es2020',
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        entryFileNames: 'js/[name].js',
        chunkFileNames: 'js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'css/[name][extname]'
          }
          return 'assets/[name]-[hash][extname]'
        },
manualChunks: {
          // Group all vendor dependencies into logical chunks
          // Order matters: chunks listed first can be imported by later chunks
          'vendor-react': [
            'react',
            'react-dom',
            'react-router-dom',
            'scheduler',
          ],
          'vendor-ui': [
            '@mantine/core',
            '@mantine/hooks',
            '@mantine/dates',
            '@mantine/form',
            '@mantine/notifications',
            '@mantine/code-highlight',
            '@tabler/icons-react',
            '@floating-ui/react',
            'framer-motion',
            'clsx',
          ],
          'vendor-charts': [
            'recharts',
            'd3-scale',
            'd3-shape',
            'd3-path',
            'd3-interpolate',
            'd3-color',
            'd3-format',
            'd3-time',
            'd3-time-format',
            'd3-array',
          ],
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
    },
  },
}))

import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [tailwindcss()],
  root: 'src',
  build: {
    outDir: '../../static',
    emptyOutDir: true,
    rollupOptions: {
      input: 'src/index.html',
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8787',
    },
  },
});

import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    include: ['src/**/*.test.ts', 'src/__tests__/**/*.ts'],
    exclude: ['src/__tests__/fixtures/**'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.ts'],
      exclude: [
        'src/index.ts',
        'src/**/*.test.ts',
        'src/__tests__/**',
      ],
      thresholds: {
        statements: 70,
        branches: 60,
        functions: 70,
        lines: 70,
      },
    },
    testTimeout: 30000, // 30s for integration tests
  },
  resolve: {
    alias: {
      '@': './src',
      // Mock wrangler-injected modules for testing
      '__STATIC_CONTENT_MANIFEST': './src/__tests__/fixtures/mock-manifest.ts',
    },
  },
})

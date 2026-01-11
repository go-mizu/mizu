import { defineConfig, devices } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const DATA_DIR = process.env.TABLE_DATA_DIR || './.e2e-data';
const basePort = (() => {
  try {
    const url = new URL(BASE_URL);
    return url.port || '8080';
  } catch {
    return '8080';
  }
})();

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  timeout: 60000,
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: `cd .. && rm -rf ${DATA_DIR} && GOWORK=off go run . --data ${DATA_DIR} seed && GOWORK=off go run . --data ${DATA_DIR} serve --dev --addr :${basePort}`,
    url: BASE_URL,
    reuseExistingServer: false,
    timeout: 120000,
  },
});

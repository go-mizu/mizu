import { test as teardown } from '@playwright/test';

teardown('cleanup', async () => {
  // Cleanup is handled by the test server
  // Additional cleanup can be added here if needed
  console.log('E2E tests completed');
});

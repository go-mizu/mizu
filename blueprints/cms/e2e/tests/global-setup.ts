import { test as setup, request } from '@playwright/test';
import { APIClient } from '../utils/api-client';
import { seedTestData, defaultTestData } from '../utils/test-data';

setup('seed test data', async () => {
  const apiContext = await request.newContext({
    baseURL: process.env.CMS_TEST_URL || 'http://localhost:8080',
  });

  const api = new APIClient(apiContext);

  // Seed test data
  await seedTestData(api, defaultTestData);

  // Login and save session for reuse
  const { session } = await api.login(defaultTestData.adminUser.email, defaultTestData.adminUser.password);

  // Store session for use in tests
  process.env.TEST_SESSION = session;

  await apiContext.dispose();
});

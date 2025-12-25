import { test as base, expect, Page } from '@playwright/test';

// Test user data
export const testUsers = {
  alice: {
    username: 'alice',
    displayName: 'Alice Smith',
    email: 'alice@example.com',
    password: 'password123',
  },
  bob: {
    username: 'bob',
    displayName: 'Bob Jones',
    email: 'bob@example.com',
    password: 'password123',
  },
  charlie: {
    username: 'charlie',
    displayName: 'Charlie Brown',
    email: 'charlie@example.com',
    password: 'password123',
  },
};

export type TestUser = keyof typeof testUsers;

// Extended test fixture with authentication helpers
export const test = base.extend<{
  authenticatedPage: Page;
  loginAs: (user: TestUser) => Promise<void>;
  registerUser: (data: {
    username: string;
    displayName?: string;
    email?: string;
    password: string;
  }) => Promise<void>;
}>({
  authenticatedPage: async ({ page }, use) => {
    // Login as alice by default
    await page.goto('/login');
    await page.fill('#login', testUsers.alice.username);
    await page.fill('#password', testUsers.alice.password);
    await page.click('button[type="submit"]');
    await page.waitForURL('/app');
    await use(page);
  },

  loginAs: async ({ page }, use) => {
    const loginAs = async (user: TestUser) => {
      const userData = testUsers[user];
      await page.goto('/login');
      await page.fill('#login', userData.username);
      await page.fill('#password', userData.password);
      await page.click('button[type="submit"]');
      await page.waitForURL('/app');
    };
    await use(loginAs);
  },

  registerUser: async ({ page }, use) => {
    const registerUser = async (data: {
      username: string;
      displayName?: string;
      email?: string;
      password: string;
    }) => {
      await page.goto('/register');
      if (data.displayName) {
        await page.fill('#display_name', data.displayName);
      }
      await page.fill('#username', data.username);
      if (data.email) {
        await page.fill('#email', data.email);
      }
      await page.fill('#password', data.password);
      await page.click('button[type="submit"]');
    };
    await use(registerUser);
  },
});

export { expect };

// Helper to generate unique test data
export function generateTestUsername(): string {
  return `testuser_${Date.now()}_${Math.random().toString(36).substring(7)}`;
}

// Helper to wait for WebSocket connection
export async function waitForWebSocket(page: Page): Promise<void> {
  await page.waitForFunction(() => {
    // Check if WebSocket is connected (look for ws variable in page context)
    return (window as any).ws?.readyState === WebSocket.OPEN;
  }, { timeout: 10000 });
}

// Helper to intercept and mock API responses
export async function mockApiResponse(
  page: Page,
  url: string | RegExp,
  response: object,
  status = 200
): Promise<void> {
  await page.route(url, (route) => {
    route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(response),
    });
  });
}

import { test as base, expect, Page } from '@playwright/test';

// Test user data matching seeded data
export const testUsers = {
  alice: {
    username: 'alice',
    email: 'alice@example.com',
    password: 'password123',
    displayName: 'Alice Johnson',
  },
  bob: {
    username: 'bob',
    email: 'bob@example.com',
    password: 'password123',
    displayName: 'Bob Smith',
  },
  charlie: {
    username: 'charlie',
    email: 'charlie@example.com',
    password: 'password123',
    displayName: 'Charlie Brown',
  },
  diana: {
    username: 'diana',
    email: 'diana@example.com',
    password: 'password123',
    displayName: 'Diana Prince',
  },
  eve: {
    username: 'eve',
    email: 'eve@example.com',
    password: 'password123',
    displayName: 'Eve Wilson',
  },
};

export type TestUser = keyof typeof testUsers;

// Generate unique test data
export function generateTestEmail(): string {
  return `test_${Date.now()}_${Math.random().toString(36).substring(7)}@example.com`;
}

export function generateTestUsername(): string {
  return `testuser_${Date.now()}_${Math.random().toString(36).substring(7)}`;
}

// Custom fixtures
type TestFixtures = {
  authenticatedPage: Page;
  loginAs: (user: TestUser) => Promise<void>;
  registerUser: (data: { username: string; email: string; displayName: string; password: string }) => Promise<void>;
};

export const test = base.extend<TestFixtures>({
  // Pre-authenticated page (as alice)
  authenticatedPage: async ({ page }, use) => {
    await page.goto('/login');
    await page.fill('#email', testUsers.alice.email);
    await page.fill('#password', testUsers.alice.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(app|acme)/);
    await use(page);
  },

  // Login as any test user
  loginAs: async ({ page }, use) => {
    const loginAs = async (user: TestUser) => {
      const userData = testUsers[user];
      await page.goto('/login');
      await page.fill('#email', userData.email);
      await page.fill('#password', userData.password);
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/(app|acme)/);
    };
    await use(loginAs);
  },

  // Register a new user
  registerUser: async ({ page }, use) => {
    const registerUser = async (data: { username: string; email: string; displayName: string; password: string }) => {
      await page.goto('/register');
      await page.fill('#username', data.username);
      await page.fill('#email', data.email);
      await page.fill('#display_name', data.displayName);
      await page.fill('#password', data.password);
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/(app|acme)/);
    };
    await use(registerUser);
  },
});

export { expect };

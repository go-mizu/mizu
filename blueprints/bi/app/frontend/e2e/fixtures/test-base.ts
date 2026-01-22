import { test as base, expect, Page } from '@playwright/test';
import path from 'path';

/**
 * Custom test fixture with authentication and common utilities
 */

// Test user credentials
export const TEST_USER = {
  email: 'admin@example.com',
  password: 'admin',
  name: 'Admin User',
  role: 'admin',
};

// Screenshot settings
const SCREENSHOT_DIR = path.join(__dirname, '../screenshots');

// Extend base test with custom fixtures
export const test = base.extend<{
  authenticatedPage: Page;
  screenshot: (name: string) => Promise<void>;
}>({
  // Authenticated page fixture - logs in before each test
  authenticatedPage: async ({ page }, use) => {
    // Navigate to login
    await page.goto('/');

    // Check if already logged in
    const isLoggedIn = await page.locator('text=Home').isVisible({ timeout: 2000 }).catch(() => false);

    if (!isLoggedIn) {
      // Wait for login form
      await page.waitForSelector('input[placeholder*="email" i], input[type="email"]', { timeout: 10000 });

      // Fill login form
      await page.fill('input[placeholder*="email" i], input[type="email"]', TEST_USER.email);
      await page.fill('input[placeholder*="password" i], input[type="password"]', TEST_USER.password);

      // Submit login
      await page.click('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")');

      // Wait for navigation
      await page.waitForURL('**/');
    }

    // Use the authenticated page
    await use(page);
  },

  // Screenshot helper fixture
  screenshot: async ({ page }, use) => {
    const takeScreenshot = async (name: string) => {
      const sanitizedName = name.replace(/[^a-zA-Z0-9_-]/g, '_');
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
      const filename = `${sanitizedName}_${timestamp}.png`;
      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, filename),
        fullPage: true,
      });
    };
    await use(takeScreenshot);
  },
});

export { expect };

/**
 * Common test utilities
 */

// Wait for API response
export async function waitForAPI(page: Page, urlPattern: string | RegExp) {
  return page.waitForResponse(
    response => {
      if (typeof urlPattern === 'string') {
        return response.url().includes(urlPattern);
      }
      return urlPattern.test(response.url());
    },
    { timeout: 30000 }
  );
}

// Wait for loading to complete
export async function waitForLoading(page: Page) {
  // Wait for any loading indicators to disappear
  await page.waitForSelector('[data-loading="true"]', { state: 'hidden', timeout: 30000 }).catch(() => {});
  await page.waitForSelector('.mantine-Loader', { state: 'hidden', timeout: 30000 }).catch(() => {});
  // Small delay for animations
  await page.waitForTimeout(300);
}

// Compare UI to Metabase reference
export async function compareToMetabase(page: Page, referenceName: string) {
  // Take screenshot for comparison
  const screenshot = await page.screenshot({ fullPage: true });

  // In a real implementation, this would compare against reference images
  // For now, we just save the screenshot
  return screenshot;
}

// Get all visible text on page
export async function getPageText(page: Page) {
  return page.evaluate(() => document.body.innerText);
}

// Check for console errors
export async function checkNoConsoleErrors(page: Page) {
  const errors: string[] = [];
  page.on('console', msg => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });
  return errors;
}

// Navigate with retry
export async function navigateWithRetry(page: Page, url: string, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      await page.goto(url, { waitUntil: 'networkidle' });
      return;
    } catch (error) {
      if (i === retries - 1) throw error;
      await page.waitForTimeout(1000);
    }
  }
}

// Fill form field with label
export async function fillField(page: Page, label: string, value: string) {
  const field = page.locator(`label:has-text("${label}") + input, label:has-text("${label}") ~ input, label:has-text("${label}") input`);
  await field.fill(value);
}

// Click button with text
export async function clickButton(page: Page, text: string) {
  await page.click(`button:has-text("${text}")`);
}

// Select from dropdown
export async function selectOption(page: Page, label: string, value: string) {
  await page.click(`label:has-text("${label}") + div, label:has-text("${label}") ~ div`);
  await page.click(`[role="option"]:has-text("${value}")`);
}

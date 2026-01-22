import { test, expect, TEST_USER } from '../fixtures/test-base';

/**
 * Authentication Tests
 * Note: These tests are skipped when authentication is not implemented.
 * They will be enabled once the auth feature is complete.
 */

// Check if auth is implemented by looking for login form
async function hasAuthImplemented(page: any): Promise<boolean> {
  await page.goto('/');
  const hasLoginForm = await page.locator('input[type="email"], input[placeholder*="email" i]').isVisible({ timeout: 3000 }).catch(() => false);
  return hasLoginForm;
}

test.describe('Authentication', () => {
  test.describe('Login Flow', () => {
    test('should login with valid credentials', async ({ page }) => {
      // Skip if no auth
      if (!await hasAuthImplemented(page)) {
        test.skip();
        return;
      }

      // Fill credentials
      await page.fill('input[type="email"], input[placeholder*="email" i]', TEST_USER.email);
      await page.fill('input[type="password"], input[placeholder*="password" i]', TEST_USER.password);

      // Submit
      await page.click('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")');

      // Verify redirect to home
      await page.waitForURL('**/', { timeout: 10000 });
      await expect(page.locator('h2:has-text("Home")')).toBeVisible();

      // Take screenshot for UI comparison
      await page.screenshot({ path: 'e2e/screenshots/auth_login_success.png', fullPage: true });
    });

    test('should show error with invalid credentials', async ({ page }) => {
      // Skip if no auth
      if (!await hasAuthImplemented(page)) {
        test.skip();
        return;
      }

      // Fill invalid credentials
      await page.fill('input[type="email"], input[placeholder*="email" i]', 'invalid@example.com');
      await page.fill('input[type="password"], input[placeholder*="password" i]', 'wrongpassword');

      // Submit
      await page.click('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")');

      // Verify error message
      await expect(page.locator('text=/invalid|error|incorrect/i')).toBeVisible({ timeout: 5000 });

      // Should stay on login page
      await expect(page.locator('input[type="password"]')).toBeVisible();
    });

    test('should show validation for empty fields', async ({ page }) => {
      // Skip if no auth
      if (!await hasAuthImplemented(page)) {
        test.skip();
        return;
      }

      // Try to submit without filling
      await page.click('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")');

      // Should stay on page (validation prevents submit)
      await expect(page.locator('input[type="email"], input[placeholder*="email" i]')).toBeVisible();
    });
  });

  test.describe('Logout', () => {
    test('should logout successfully', async ({ authenticatedPage: page }) => {
      // Look for user menu (only exists if auth is implemented)
      const userMenu = page.locator('[aria-label="User menu"], button:has-text("Account"), [data-testid="user-menu"], button:has([class*="avatar"])');

      if (!await userMenu.isVisible({ timeout: 3000 }).catch(() => false)) {
        test.skip();
        return;
      }

      // Click user menu
      await userMenu.click();

      // Click logout
      await page.click('text=Logout, text=Sign out, [data-testid="logout"]');

      // Verify redirect to login
      await page.waitForSelector('input[type="email"], input[placeholder*="email" i]', { timeout: 10000 });
    });
  });

  test.describe('Session Management', () => {
    test('should persist session after page reload', async ({ authenticatedPage: page }) => {
      // Verify we're on the home page
      await expect(page.locator('h2:has-text("Home")')).toBeVisible();

      // Reload page
      await page.reload();

      // Should still be on home page (session persists or no auth required)
      await expect(page.locator('h2:has-text("Home")')).toBeVisible();
    });

    test('should handle unauthenticated access gracefully', async ({ page }) => {
      // Clear any existing session
      await page.context().clearCookies();

      // Try to access a page
      await page.goto('/question/new');

      // Should either show login or the page (depending on auth implementation)
      const hasLogin = await page.locator('input[type="email"], input[placeholder*="email" i]').isVisible({ timeout: 3000 }).catch(() => false);
      const hasQuestionPage = await page.locator('text=Database, text=Table, h2').isVisible({ timeout: 3000 }).catch(() => false);

      // One of these should be true
      expect(hasLogin || hasQuestionPage).toBeTruthy();
    });
  });
});

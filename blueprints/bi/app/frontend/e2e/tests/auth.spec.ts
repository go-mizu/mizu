import { test, expect, TEST_USER } from '../fixtures/test-base';

test.describe('Authentication', () => {
  test.describe('Login Flow', () => {
    test('should login with valid credentials', async ({ page }) => {
      await page.goto('/');

      // Wait for login page
      await page.waitForSelector('input[type="email"], input[placeholder*="email" i]');

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
      await page.goto('/');

      // Wait for login page
      await page.waitForSelector('input[type="email"], input[placeholder*="email" i]');

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
      await page.goto('/');

      // Wait for login page
      await page.waitForSelector('input[type="email"], input[placeholder*="email" i]');

      // Try to submit without filling
      await page.click('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")');

      // Should stay on page (validation prevents submit)
      await expect(page.locator('input[type="email"], input[placeholder*="email" i]')).toBeVisible();
    });
  });

  test.describe('Logout', () => {
    test('should logout successfully', async ({ authenticatedPage: page }) => {
      // Find and click logout
      await page.click('[aria-label="User menu"], button:has-text("Account"), [data-testid="user-menu"]').catch(async () => {
        // Try alternative selectors
        await page.click('button:has([class*="avatar"]), [data-testid="avatar-button"]');
      });

      await page.click('text=Logout, text=Sign out, [data-testid="logout"]');

      // Verify redirect to login
      await page.waitForSelector('input[type="email"], input[placeholder*="email" i]', { timeout: 10000 });
    });
  });

  test.describe('Session Management', () => {
    test('should persist session after page reload', async ({ authenticatedPage: page }) => {
      // Verify we're logged in
      await expect(page.locator('h2:has-text("Home")')).toBeVisible();

      // Reload page
      await page.reload();

      // Should still be logged in
      await expect(page.locator('h2:has-text("Home")')).toBeVisible();
    });

    test('should redirect protected routes to login when not authenticated', async ({ page }) => {
      // Clear any existing session
      await page.context().clearCookies();

      // Try to access protected route
      await page.goto('/question/new');

      // Should redirect to login
      await expect(page.locator('input[type="email"], input[placeholder*="email" i], text=Sign in')).toBeVisible({ timeout: 10000 });
    });
  });
});

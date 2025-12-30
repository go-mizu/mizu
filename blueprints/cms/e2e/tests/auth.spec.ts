import { test, expect, URLS, TEST_ADMIN } from '../fixtures/test-fixtures';
import { LoginPage } from '../pages/login.page';

test.describe('Authentication', () => {
  test.describe('Login Page', () => {
    test('login page renders correctly', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.expectLoginForm();
    });

    test('login page renders with legacy URL', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.gotoLegacy();
      await loginPage.expectLoginForm();
    });

    test('successful login redirects to dashboard', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.login(TEST_ADMIN.email, TEST_ADMIN.password);
      await loginPage.expectRedirectToDashboard();
    });

    test('failed login with invalid password shows error', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.login(TEST_ADMIN.email, 'wrongpassword');
      await loginPage.expectError();
    });

    test('failed login with unknown user shows error', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.login('unknown@test.com', 'password');
      await loginPage.expectError();
    });
  });

  test.describe('Protected Routes', () => {
    test('dashboard redirects to login when not authenticated', async ({ page }) => {
      await page.goto(URLS.dashboard);
      await expect(page).toHaveURL(/login|wp-login/);
    });

    test('posts page redirects to login when not authenticated', async ({ page }) => {
      await page.goto(URLS.posts);
      await expect(page).toHaveURL(/login|wp-login/);
    });

    test('users page redirects to login when not authenticated', async ({ page }) => {
      await page.goto(URLS.users);
      await expect(page).toHaveURL(/login|wp-login/);
    });

    test('settings page redirects to login when not authenticated', async ({ page }) => {
      await page.goto(URLS.settingsGeneral);
      await expect(page).toHaveURL(/login|wp-login/);
    });
  });

  test.describe('Session Management', () => {
    test('authenticated user can access dashboard', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.dashboard);
      await expect(authenticatedPage).toHaveURL(/\/wp-admin\/?$/);
    });

    test('session persists after page refresh', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.dashboard);
      await authenticatedPage.reload();
      await expect(authenticatedPage).toHaveURL(/\/wp-admin\/?$/);
    });

    test('login with redirect_to parameter', async ({ page }) => {
      // Go to protected page first
      await page.goto(URLS.posts);

      // Should redirect to login with redirect_to param
      const loginUrl = page.url();
      expect(loginUrl).toContain('redirect_to');

      // Login
      const loginPage = new LoginPage(page);
      await loginPage.login(TEST_ADMIN.email, TEST_ADMIN.password);

      // Should redirect back to posts page
      await expect(page).toHaveURL(/posts|edit\.php/);
    });
  });

  test.describe('Clean URLs vs Legacy URLs', () => {
    test('clean login URL works', async ({ page }) => {
      await page.goto(URLS.login);
      await expect(page.locator('form')).toBeVisible();
    });

    test('legacy login URL works', async ({ page }) => {
      await page.goto(URLS.legacy.login);
      await expect(page.locator('form')).toBeVisible();
    });
  });
});

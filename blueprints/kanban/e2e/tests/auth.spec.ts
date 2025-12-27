import { test, expect, testUsers, generateTestEmail, generateTestUsername } from '../fixtures/test-fixtures.js';
import { LoginPage } from '../pages/login.page.js';
import { RegisterPage } from '../pages/register.page.js';
import { HomePage } from '../pages/home.page.js';

test.describe('Authentication Flow', () => {
  test.describe('User Registration', () => {
    test('TC-AUTH-001: successful registration with all fields', async ({ page }) => {
      const registerPage = new RegisterPage(page);
      await registerPage.goto();
      await registerPage.expectToBeOnRegisterPage();

      const email = generateTestEmail();
      const username = generateTestUsername();

      await registerPage.register({
        username,
        email,
        displayName: 'Test User',
        password: 'password123',
      });

      await registerPage.expectSuccessfulRegistration();
    });

    test('TC-AUTH-002: registration fails with existing email', async ({ page }) => {
      const registerPage = new RegisterPage(page);
      await registerPage.goto();

      await registerPage.register({
        username: generateTestUsername(),
        email: testUsers.alice.email, // Already exists
        displayName: 'Test User',
        password: 'password123',
      });

      await registerPage.expectError();
    });

    test('TC-AUTH-003: registration fails with short password', async ({ page }) => {
      const registerPage = new RegisterPage(page);
      await registerPage.goto();

      await registerPage.register({
        username: generateTestUsername(),
        email: generateTestEmail(),
        displayName: 'Test User',
        password: 'short',
      });

      // Form validation should prevent submission or show error
      await expect(page).toHaveURL(/register/);
    });
  });

  test.describe('User Login', () => {
    test('TC-AUTH-004: successful login with email', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.expectToBeOnLoginPage();

      await loginPage.login(testUsers.alice.email, testUsers.alice.password);

      await loginPage.expectSuccessfulLogin();
    });

    test('TC-AUTH-005: login fails with wrong password', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();

      await loginPage.login(testUsers.alice.email, 'wrongpassword');

      await loginPage.expectError();
    });

    test('TC-AUTH-006: login fails with non-existent user', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();

      await loginPage.login('nonexistent@example.com', 'password123');

      await loginPage.expectError();
    });

    test('TC-AUTH-007: login page has link to register', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();

      await expect(loginPage.registerLink).toBeVisible();
      await loginPage.registerLink.click();

      await expect(page).toHaveURL(/register/);
    });
  });

  test.describe('Logout', () => {
    test('TC-AUTH-008: logout redirects to login page', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      await homePage.logout();

      await expect(page).toHaveURL(/login/);
    });
  });

  test.describe('Protected Routes', () => {
    test('TC-AUTH-009: protected pages redirect unauthenticated users', async ({ page }) => {
      // Try to access protected route without login
      await page.goto('/w/acme');

      // Should redirect to login
      await expect(page).toHaveURL(/login/);
    });

    test('TC-AUTH-010: accessing board without auth redirects to login', async ({ page }) => {
      await page.goto('/w/acme/board/some-project');

      await expect(page).toHaveURL(/login/);
    });

    test('TC-AUTH-011: accessing issues without auth redirects to login', async ({ page }) => {
      await page.goto('/w/acme/issues');

      await expect(page).toHaveURL(/login/);
    });
  });
});

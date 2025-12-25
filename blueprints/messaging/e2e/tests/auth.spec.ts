import { test, expect, generateTestUsername, testUsers } from '../fixtures/test-fixtures';
import { LoginPage } from '../pages/login.page';
import { RegisterPage } from '../pages/register.page';
import { AppPage } from '../pages/app.page';

test.describe('Authentication Flow', () => {
  test.describe('User Registration', () => {
    test('TC-AUTH-001: successful registration with all fields', async ({ page }) => {
      const registerPage = new RegisterPage(page);
      const appPage = new AppPage(page);

      await registerPage.goto();
      await registerPage.expectHeading();

      const username = generateTestUsername();
      await registerPage.register({
        displayName: 'Test User',
        username,
        email: 'test@example.com',
        password: 'password123',
      });

      await registerPage.expectSuccessfulRegistration();
      await appPage.waitForLoad();
      await expect(appPage.userName).toContainText('Test User');
    });

    test('TC-AUTH-001b: successful registration with minimal fields', async ({ page }) => {
      const registerPage = new RegisterPage(page);

      await registerPage.goto();

      const username = generateTestUsername();
      await registerPage.register({
        username,
        password: 'password123',
      });

      await registerPage.expectSuccessfulRegistration();
    });

    test('TC-AUTH-002: registration fails with duplicate username', async ({ page, registerUser }) => {
      const registerPage = new RegisterPage(page);

      // First registration
      const username = generateTestUsername();
      await registerUser({
        username,
        password: 'password123',
      });

      // Logout and try again
      await page.goto('/');

      await registerPage.goto();
      await registerPage.register({
        username,
        password: 'password123',
      });

      await registerPage.expectErrorMessage();
    });

    test('TC-AUTH-003a: registration requires username', async ({ page }) => {
      const registerPage = new RegisterPage(page);

      await registerPage.goto();
      await registerPage.passwordInput.fill('password123');
      await registerPage.submitButton.click();

      // HTML5 validation should prevent submission
      const usernameInput = registerPage.usernameInput;
      await expect(usernameInput).toHaveAttribute('required', '');
    });

    test('TC-AUTH-003b: registration requires minimum password length', async ({ page }) => {
      const registerPage = new RegisterPage(page);

      await registerPage.goto();
      await registerPage.usernameInput.fill(generateTestUsername());
      await registerPage.passwordInput.fill('12345');
      await registerPage.submitButton.click();

      // HTML5 validation should prevent submission with short password
      await expect(registerPage.passwordInput).toHaveAttribute('minlength', '6');
    });

    test('can navigate from register to login', async ({ page }) => {
      const registerPage = new RegisterPage(page);

      await registerPage.goto();
      await registerPage.loginLink.click();
      await page.waitForURL('/login');
    });
  });

  test.describe('User Login', () => {
    test('TC-AUTH-004: successful login with username', async ({ page }) => {
      const loginPage = new LoginPage(page);
      const appPage = new AppPage(page);

      await loginPage.goto();
      await loginPage.expectHeading();

      await loginPage.login(testUsers.alice.username, testUsers.alice.password);

      await loginPage.expectSuccessfulLogin();
      await appPage.waitForLoad();
      await expect(appPage.userName).not.toBeEmpty();
    });

    test('TC-AUTH-004b: successful login with email', async ({ page }) => {
      const loginPage = new LoginPage(page);

      // First register a user with email
      const registerPage = new RegisterPage(page);
      const username = generateTestUsername();
      const email = `${username}@example.com`;

      await registerPage.goto();
      await registerPage.register({
        username,
        email,
        password: 'password123',
      });

      await page.waitForURL('/app');

      // Logout
      await page.goto('/');

      // Login with email
      await loginPage.goto();
      await loginPage.login(email, 'password123');
      await loginPage.expectSuccessfulLogin();
    });

    test('TC-AUTH-005: login fails with wrong password', async ({ page }) => {
      const loginPage = new LoginPage(page);

      await loginPage.goto();
      await loginPage.login(testUsers.alice.username, 'wrongpassword');

      await loginPage.expectErrorMessage();
      await expect(page).toHaveURL('/login');
    });

    test('TC-AUTH-006: login fails for non-existent user', async ({ page }) => {
      const loginPage = new LoginPage(page);

      await loginPage.goto();
      await loginPage.login('nonexistent_user_12345_xyz', 'anypassword');

      await loginPage.expectErrorMessage();
    });

    test('can navigate from login to register', async ({ page }) => {
      const loginPage = new LoginPage(page);

      await loginPage.goto();
      await loginPage.registerLink.click();
      await page.waitForURL('/register');
    });
  });

  test.describe('Session Management', () => {
    test('TC-AUTH-007: session persists after page refresh', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      const userNameBefore = await appPage.userName.textContent();

      // Refresh the page
      await page.reload();

      await appPage.waitForLoad();
      const userNameAfter = await appPage.userName.textContent();

      expect(userNameAfter).toBe(userNameBefore);
    });

    test('TC-AUTH-008: logout redirects to home', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Navigate to settings and logout
      await page.goto('/settings');
      await page.click('button:has-text("Log Out")');

      await page.waitForURL('/');
    });

    test('TC-AUTH-009: protected routes redirect to login', async ({ page }) => {
      // Clear any existing session
      await page.context().clearCookies();

      // Try to access protected route
      await page.goto('/app');

      // Should redirect to login
      await page.waitForURL('/login');
    });

    test('protected settings page redirects to login', async ({ page }) => {
      await page.context().clearCookies();
      await page.goto('/settings');
      await page.waitForURL('/login');
    });
  });

  test.describe('Navigation', () => {
    test('home page shows login and register links', async ({ page }) => {
      await page.goto('/');

      await expect(page.locator('a[href="/login"]')).toBeVisible();
      await expect(page.locator('a[href="/register"]')).toBeVisible();
    });

    test('home page Get Started button goes to register', async ({ page }) => {
      await page.goto('/');

      await page.click('a:has-text("Get Started")');
      await page.waitForURL('/register');
    });

    test('home page Start Messaging button goes to register', async ({ page }) => {
      await page.goto('/');

      await page.click('a:has-text("Start Messaging")');
      await page.waitForURL('/register');
    });
  });
});

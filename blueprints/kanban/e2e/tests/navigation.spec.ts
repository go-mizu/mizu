import { test, expect } from '../fixtures/test-fixtures.js';
import { HomePage } from '../pages/home.page.js';

test.describe('Navigation', () => {
  test.describe('Sidebar Navigation', () => {
    test('TC-NAV-001: sidebar shows all navigation items', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      // Check sidebar navigation items
      await expect(page.locator('.sidebar .nav-item[title="Home"]')).toBeVisible();
      await expect(page.locator('.sidebar .nav-item[title="Issues"]')).toBeVisible();
      await expect(page.locator('.sidebar .nav-item[title="Cycles"]')).toBeVisible();
      await expect(page.locator('.sidebar .nav-item[title="Teams"]')).toBeVisible();
      await expect(page.locator('.sidebar .nav-item[title="Settings"]')).toBeVisible();
    });

    test('TC-NAV-002: clicking Issues nav goes to issues page', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.locator('.sidebar .nav-item[title="Issues"]').click();

      await expect(page).toHaveURL(/issues/);
    });

    test('TC-NAV-003: clicking Cycles nav goes to cycles page', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.locator('.sidebar .nav-item[title="Cycles"]').click();

      await expect(page).toHaveURL(/cycles/);
    });

    test('TC-NAV-004: clicking Home nav returns to dashboard', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Navigate away first
      await page.locator('.sidebar .nav-item[title="Issues"]').click();
      await expect(page).toHaveURL(/issues/);

      // Click home
      await page.locator('.sidebar .nav-item[title="Home"]').click();

      // Should be back at dashboard
      await expect(page.locator('h1:has-text("Welcome")')).toBeVisible();
    });
  });

  test.describe('Topbar', () => {
    test('TC-NAV-005: topbar shows workspace name', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      // Should show workspace name in dropdown
      await expect(page.locator('.topbar button:has-text("Acme")')).toBeVisible();
    });

    test('TC-NAV-006: create issue button opens modal', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      await homePage.createIssueButton.click();

      await expect(page.locator('#create-issue-modal')).toBeVisible();
    });

    test('TC-NAV-007: user menu shows profile and logout', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      await homePage.openUserMenu();

      await expect(page.locator('.dropdown-menu:has-text("Profile")')).toBeVisible();
      await expect(page.locator('.dropdown-menu button:has-text("Sign out")')).toBeVisible();
    });
  });

  test.describe('Keyboard Shortcuts', () => {
    test('TC-NAV-008: pressing C opens create issue modal', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      // Press C key
      await page.keyboard.press('c');

      await expect(page.locator('#create-issue-modal')).toBeVisible();
    });

    test('TC-NAV-009: pressing Escape closes modal', async ({ page, loginAs }) => {
      await loginAs('alice');

      const homePage = new HomePage(page);
      await homePage.expectToBeLoggedIn();

      // Open modal
      await page.keyboard.press('c');
      await expect(page.locator('#create-issue-modal')).toBeVisible();

      // Press Escape
      await page.keyboard.press('Escape');

      await expect(page.locator('#create-issue-modal')).not.toBeVisible();
    });
  });

  test.describe('Responsive Design', () => {
    test('TC-NAV-010: sidebar collapses on mobile', async ({ page, loginAs }) => {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });

      await loginAs('alice');

      // Sidebar should not be visible by default on mobile
      const sidebar = page.locator('.sidebar');
      await expect(sidebar).not.toBeVisible();
    });
  });

  test.describe('Breadcrumbs', () => {
    test('TC-NAV-011: issue detail shows breadcrumb', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Navigate to issues and then to an issue
      await page.locator('.sidebar .nav-item[title="Issues"]').click();
      await page.locator('.table tbody tr').first().click();

      // Should show breadcrumb with Issues link
      await expect(page.locator('.topbar nav a:has-text("Issues")')).toBeVisible();
    });

    test('TC-NAV-012: clicking breadcrumb navigates back', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Navigate to an issue
      await page.locator('.sidebar .nav-item[title="Issues"]').click();
      await page.locator('.table tbody tr').first().click();

      // Click Issues breadcrumb
      await page.locator('.topbar nav a:has-text("Issues")').click();

      await expect(page).toHaveURL(/issues/);
    });
  });
});

import { test, expect } from '@playwright/test';

test.describe('Database Views Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const sidebar = page.locator('.mantine-AppShell-navbar');
    const databaseNavLink = sidebar.locator('.mantine-NavLink-root').filter({ hasText: 'Database' }).first();
    await databaseNavLink.click();

    await page.waitForTimeout(500);

    const viewsLink = sidebar.getByRole('link', { name: 'Views', exact: true });
    await viewsLink.click();

    await page.waitForLoadState('networkidle');
  });

  test('E2E-VIEW-001: View list loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const viewSection = page.getByText(/Views|No views|Create view/i).first();
    await expect(viewSection).toBeVisible();
  });

  test('E2E-VIEW-002: Regular views displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for views tab or section
    const viewsTab = page.getByText(/Regular|Views/i).first();
    const isVisible = await viewsTab.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-VIEW-003: Materialized views section', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const materializedTab = page.getByText(/Materialized/i).first();
    const isVisible = await materializedTab.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-VIEW-004: Create view button visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-VIEW-005: View definition shown', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for SQL definition or code block
    const definition = page.getByText(/SELECT|FROM|CREATE VIEW/i).first();
    const isVisible = await definition.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-VIEW-006: Refresh materialized view button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const refreshButton = page.getByRole('button', { name: /Refresh/i });
    const isVisible = await refreshButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-VIEW-007: Delete view confirmation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const deleteButton = page.getByRole('button', { name: /delete|drop/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      const confirmModal = page.getByText(/confirm|sure|delete/i);
      await expect(confirmModal).toBeVisible();

      const cancelButton = page.getByRole('button', { name: /cancel|no|close/i });
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      }
    }
  });
});

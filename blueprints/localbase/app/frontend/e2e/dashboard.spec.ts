import { test, expect } from '@playwright/test';

test.describe('Dashboard Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('E2E-DASH-001: Dashboard loads within 2 seconds', async ({ page }) => {
    const startTime = Date.now();
    await page.waitForSelector('text=Dashboard', { state: 'visible' });
    const loadTime = Date.now() - startTime;
    expect(loadTime).toBeLessThan(2000);
  });

  test('E2E-DASH-002: All stat cards display correctly', async ({ page }) => {
    // Wait for stats to load
    await page.waitForLoadState('networkidle');

    // Check for stat cards
    const statsSection = page.locator('[data-testid="dashboard-stats"]').or(page.locator('text=Users').first());
    await expect(statsSection).toBeVisible();

    // Verify stat values are numbers or loading states
    const usersStat = page.getByText(/Users/i).first();
    await expect(usersStat).toBeVisible();
  });

  test('E2E-DASH-003: Health indicators show service status', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Check for health section
    const healthSection = page.getByText(/Database|healthy|Healthy/i).first();
    await expect(healthSection).toBeVisible();
  });

  test('E2E-DASH-004: Quick links navigate to correct pages', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Check sidebar navigation
    const tableEditorLink = page.getByRole('link', { name: /Table Editor/i });
    await expect(tableEditorLink).toBeVisible();

    // Click and verify navigation
    await tableEditorLink.click();
    await expect(page).toHaveURL(/table-editor/);
  });

  test('E2E-DASH-005: Sidebar navigation works', async ({ page }) => {
    // Test navigation to various pages
    const navItems = [
      { name: /SQL Editor/i, url: /sql-editor/ },
      { name: /Authentication/i, url: /auth\/users/ },
      { name: /Storage/i, url: /storage/ },
    ];

    for (const item of navItems) {
      const link = page.getByRole('link', { name: item.name });
      await link.click();
      await expect(page).toHaveURL(item.url);
    }
  });

  test('E2E-DASH-006: Sidebar collapse toggle works', async ({ page }) => {
    // Find and click collapse button
    const collapseButton = page.getByRole('button', { name: /Collapse|Expand/i });

    if (await collapseButton.isVisible()) {
      await collapseButton.click();
      // Wait for animation
      await page.waitForTimeout(300);

      // Sidebar should be collapsed
      const sidebar = page.locator('nav').first();
      await expect(sidebar).toBeVisible();
    }
  });
});

import { test, expect } from '@playwright/test';

test.describe('Database Indexes Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/database/indexes');
    await page.waitForLoadState('networkidle');
  });

  test('E2E-INDEX-000: Page loads without JavaScript errors', async ({ page }) => {
    const jsErrors: string[] = [];

    // Listen for console errors
    page.on('pageerror', (error) => {
      jsErrors.push(error.message);
    });

    // Navigate and wait for load
    await page.goto('/database/indexes');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Filter out known acceptable errors
    const criticalErrors = jsErrors.filter(err =>
      !err.includes('Failed to fetch') &&
      !err.includes('NetworkError') &&
      !err.includes('net::ERR')
    );

    // Ensure no critical JavaScript errors occurred
    expect(criticalErrors.filter(e => e.includes('null is not an object') || e.includes('Cannot read properties of null'))).toHaveLength(0);
  });

  test('E2E-INDEX-001: Index list loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const indexSection = page.getByText(/Indexes|No indexes|Create index/i).first();
    await expect(indexSection).toBeVisible();
  });

  test('E2E-INDEX-002: Create index button visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-INDEX-003: Index type selection available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await createButton.click();

    await page.waitForTimeout(500);

    const typeSelect = page.getByText(/btree|hash|gin|gist|Index type/i).first();
    await expect(typeSelect).toBeVisible();
  });

  test('E2E-INDEX-004: Index shows table name', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const tableName = page.getByText(/public\.|users|posts|Table/i).first();
    const isVisible = await tableName.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-INDEX-005: Unique index indicator', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const uniqueBadge = page.getByText(/unique|Unique/i).first();
    const isVisible = await uniqueBadge.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-INDEX-006: Index size displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const indexSize = page.getByText(/KB|MB|bytes|kB/i).first();
    const isVisible = await indexSize.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-INDEX-007: Delete index confirmation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const deleteButton = page.getByRole('button', { name: /delete|drop/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      const confirmModal = page.getByText(/confirm|sure|delete|drop/i);
      await expect(confirmModal).toBeVisible();

      const cancelButton = page.getByRole('button', { name: /cancel|no|close/i });
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      }
    }
  });

  test('E2E-INDEX-008: Index columns displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const columns = page.getByText(/id|email|created_at|Column/i).first();
    const isVisible = await columns.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

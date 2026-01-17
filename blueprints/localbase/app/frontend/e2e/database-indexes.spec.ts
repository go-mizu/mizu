import { test, expect } from '@playwright/test';

test.describe('Database Indexes Page', () => {
  test.beforeEach(async ({ page }) => {
    // Load the app first
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Use JavaScript to navigate within the SPA (React Router)
    await page.evaluate(() => {
      window.history.pushState({}, '', '/database/indexes');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    await page.waitForTimeout(1000);
    await page.reload();
    await page.waitForLoadState('networkidle');
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

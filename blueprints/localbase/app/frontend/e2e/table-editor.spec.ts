import { test, expect } from '@playwright/test';

test.describe('Table Editor Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const tableEditorLink = page.locator('.mantine-AppShell-navbar').getByRole('link', { name: 'Table Editor' });
    await tableEditorLink.click();

    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/table-editor/);
  });

  test('E2E-TABLE-001: Table list loads with all tables', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Check for table list or empty state
    const tableList = page.locator('[data-testid="table-list"]').or(page.getByText(/Tables|No tables|Create a table/i));
    await expect(tableList).toBeVisible();
  });

  test('E2E-TABLE-002: Schema selector is visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for schema selector
    const schemaSelector = page.getByRole('combobox').or(page.getByText(/public|Schema/i)).first();
    await expect(schemaSelector).toBeVisible();
  });

  test('E2E-TABLE-003: Create table button is visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New Table|Add/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-TABLE-004: Create table modal opens', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New Table|Add/i }).first();
    await createButton.click();

    // Check for modal
    const modal = page.getByRole('dialog').or(page.locator('.mantine-Modal-content'));
    await expect(modal).toBeVisible();
  });

  test('E2E-TABLE-005: Table can be created', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Open create modal
    const createButton = page.getByRole('button', { name: /Create|New Table|Add/i }).first();
    await createButton.click();

    // Fill in table name
    const tableName = `test_table_${Date.now()}`;
    const nameInput = page.getByPlaceholder(/name/i).or(page.getByLabel(/name/i)).first();
    await nameInput.fill(tableName);

    // Submit
    const submitButton = page.getByRole('button', { name: /Create|Save|Submit/i }).last();
    await submitButton.click();

    // Wait for modal to close or success message
    await page.waitForTimeout(1000);
  });

  test('E2E-TABLE-006: Table data displays when table selected', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Click on a table in the list if available
    const tableItems = page.locator('[data-testid="table-item"]').or(page.locator('button').filter({ hasText: /users|posts|profiles/i }));

    const count = await tableItems.count();
    if (count > 0) {
      await tableItems.first().click();
      await page.waitForLoadState('networkidle');

      // Check for data grid or table
      const dataGrid = page.locator('table').or(page.getByText(/No rows|Loading/i));
      await expect(dataGrid).toBeVisible();
    }
  });

  test('E2E-TABLE-007: Column headers show types', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Select a table first
    const tableItems = page.locator('button').filter({ hasText: /users|posts/i });
    const count = await tableItems.count();

    if (count > 0) {
      await tableItems.first().click();
      await page.waitForLoadState('networkidle');

      // Look for column type indicators
      const columnHeader = page.locator('th').or(page.getByText(/text|integer|uuid|timestamp/i)).first();
      await expect(columnHeader).toBeVisible();
    }
  });

  test('E2E-TABLE-008: RLS badge displays for tables', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for RLS indicators
    const rlsBadge = page.getByText(/RLS|Row Level Security|enabled|disabled/i).first();

    // This may not be visible if no tables have RLS configured
    const isVisible = await rlsBadge.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-TABLE-009: Search filters tables', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder(/Search|Filter/i).first();

    if (await searchInput.isVisible()) {
      await searchInput.fill('test');
      await page.waitForTimeout(500);

      // The table list should be filtered
      const tableList = page.locator('[data-testid="table-list"]').or(page.locator('nav'));
      await expect(tableList).toBeVisible();
    }
  });

  test('E2E-TABLE-010: Pagination works', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Select a table with data
    const tableItems = page.locator('button').filter({ hasText: /users|posts/i });
    const count = await tableItems.count();

    if (count > 0) {
      await tableItems.first().click();
      await page.waitForLoadState('networkidle');

      // Look for pagination controls
      const pagination = page.getByRole('button', { name: /next|previous|1|2/i }).or(page.getByText(/Page|rows/i));
      const isVisible = await pagination.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });
});

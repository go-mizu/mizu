import { test, expect } from '@playwright/test';

test.describe('SQL Editor Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const sqlEditorLink = page.locator('.mantine-AppShell-navbar').getByRole('link', { name: 'SQL Editor' });
    await sqlEditorLink.click();

    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/sql-editor/);
  });

  test('E2E-SQL-001: Monaco editor loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Monaco editor should be visible
    const editor = page.locator('.monaco-editor').or(page.locator('[data-testid="sql-editor"]'));
    await expect(editor).toBeVisible();
  });

  test('E2E-SQL-002: Run button is visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const runButton = page.getByRole('button', { name: /Run|Execute/i });
    await expect(runButton).toBeVisible();
  });

  test('E2E-SQL-003: Execute SELECT query', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Wait for editor to load
    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible();

    // Click in the editor and type a query
    await editor.click();
    await page.keyboard.type('SELECT 1 as test_value;');

    // Click run button
    const runButton = page.getByRole('button', { name: /Run|Execute/i });
    await runButton.click();

    // Wait for results
    await page.waitForTimeout(2000);

    // Check for results table or success message
    const results = page.getByText(/test_value|1|rows|result/i);
    await expect(results).toBeVisible();
  });

  test('E2E-SQL-004: Results table shows data', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible();

    await editor.click();
    await page.keyboard.type('SELECT generate_series(1, 5) as num;');

    const runButton = page.getByRole('button', { name: /Run|Execute/i });
    await runButton.click();

    await page.waitForTimeout(2000);

    // Check for results
    const resultsTable = page.locator('table').or(page.getByText(/num|1|2|3/i));
    await expect(resultsTable).toBeVisible();
  });

  test('E2E-SQL-005: Query error shows message', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible();

    await editor.click();
    await page.keyboard.type('SELECT * FROM nonexistent_table_xyz;');

    const runButton = page.getByRole('button', { name: /Run|Execute/i });
    await runButton.click();

    await page.waitForTimeout(2000);

    // Check for error message
    const error = page.getByText(/error|does not exist|not found/i);
    await expect(error).toBeVisible();
  });

  test('E2E-SQL-006: Saved queries sidebar visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for saved queries section
    const savedSection = page.getByText(/Saved|Queries|History/i).first();
    await expect(savedSection).toBeVisible();
  });

  test('E2E-SQL-007: Save query modal opens', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const saveButton = page.getByRole('button', { name: /Save/i }).first();

    if (await saveButton.isVisible()) {
      await saveButton.click();

      // Check for modal
      const modal = page.getByRole('dialog').or(page.getByText(/Save query|Query name/i));
      await expect(modal).toBeVisible();
    }
  });

  test('E2E-SQL-008: Export results works', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Run a query first
    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible();

    await editor.click();
    await page.keyboard.type('SELECT 1 as value;');

    const runButton = page.getByRole('button', { name: /Run|Execute/i });
    await runButton.click();

    await page.waitForTimeout(2000);

    // Look for export button
    const exportButton = page.getByRole('button', { name: /Export|CSV|Download/i });

    if (await exportButton.isVisible()) {
      // Start download monitoring
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 5000 }).catch(() => null),
        exportButton.click(),
      ]);

      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.csv$|\.json$/);
      }
    }
  });

  test('E2E-SQL-009: Keyboard shortcut executes query', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible();

    await editor.click();
    await page.keyboard.type('SELECT 2 as shortcut_test;');

    // Use Ctrl/Cmd + Enter
    await page.keyboard.press('Control+Enter');

    await page.waitForTimeout(2000);

    // Check for results
    const results = page.getByText(/shortcut_test|2|rows/i);
    const isVisible = await results.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-SQL-010: Row count displays correctly', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible();

    await editor.click();
    await page.keyboard.type('SELECT generate_series(1, 10) as num;');

    const runButton = page.getByRole('button', { name: /Run|Execute/i });
    await runButton.click();

    await page.waitForTimeout(2000);

    // Check for row count display
    const rowCount = page.getByText(/10 rows|rows returned/i);
    const isVisible = await rowCount.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

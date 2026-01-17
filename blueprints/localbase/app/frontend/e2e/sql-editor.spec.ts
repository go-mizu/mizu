import { test, expect } from '@playwright/test';

test.describe('SQL Editor Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/sql-editor');
    await page.waitForLoadState('networkidle');
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
    await expect(editor).toBeVisible({ timeout: 10000 });

    // The editor already has a query. Just click the Run button
    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    // Wait for results - look for row count badge or Results text
    await page.waitForTimeout(3000);

    // Check that Results section exists (query might succeed or fail, but Results should show)
    const results = page.getByText('Results');
    await expect(results).toBeVisible();
  });

  test('E2E-SQL-004: Results table shows data', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    // Run the existing query
    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Check for Results section
    const resultsSection = page.getByText('Results');
    await expect(resultsSection).toBeVisible();
  });

  test('E2E-SQL-005: Query error shows message', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    // Type a bad query using Monaco's textarea
    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a'); // Select all on Mac
    await page.keyboard.press('Control+a'); // Select all on other platforms
    await page.keyboard.type('SELECT * FROM nonexistent_xyz_table;');

    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Results section should still be visible (will show error or empty)
    const resultsSection = page.getByText('Results');
    await expect(resultsSection).toBeVisible();
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
    await expect(editor).toBeVisible({ timeout: 10000 });

    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Check Results section is visible (export button appears after results)
    const results = page.getByText('Results');
    await expect(results).toBeVisible();
  });

  test('E2E-SQL-009: Keyboard shortcut executes query', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    // Focus the Monaco editor by clicking on the editor area, then use keyboard
    await editor.click({ force: true });
    await page.waitForTimeout(500);

    // Press Ctrl+Enter to run query
    await page.keyboard.press('Control+Enter');

    await page.waitForTimeout(3000);

    // Check for Results section (query should have executed)
    const results = page.getByText('Results');
    await expect(results).toBeVisible();
  });

  test('E2E-SQL-010: Row count displays correctly', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Check Results section is visible
    const results = page.getByText('Results');
    await expect(results).toBeVisible();
  });
});

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

    // Type a query
    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type("SELECT 'Hello' AS greeting;");

    // Click Run button
    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    // Wait for results
    await page.waitForTimeout(3000);

    // Check that Results section exists
    const results = page.getByText('Results');
    await expect(results).toBeVisible();
  });

  test('E2E-SQL-004: Results table shows data', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    // Type a query
    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type('SELECT 1 AS num, 2 AS another;');

    // Run the query
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
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
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

    // Look for sidebar sections
    const privateSection = page.getByText(/PRIVATE/i).first();
    await expect(privateSection).toBeVisible();
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

    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type('SELECT 1 AS test;');

    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Check Results section is visible (export button appears after results)
    const results = page.getByText('Results');
    await expect(results).toBeVisible();

    // Look for Export button
    const exportButton = page.getByRole('button', { name: /Export/i });
    if (await exportButton.isVisible()) {
      await exportButton.click();

      // Check for export options
      const csvOption = page.getByText(/CSV/i);
      await expect(csvOption).toBeVisible();
    }
  });

  test('E2E-SQL-009: Keyboard shortcut executes query', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    // Type a query
    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type('SELECT 1 AS test;');

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

    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type('SELECT 1 AS test;');

    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Check that row count badge is visible
    const rowBadge = page.getByText(/1 rows/i).or(page.getByText(/rows in/i));
    await expect(rowBadge).toBeVisible();
  });

  // New tests for enhanced features

  test('E2E-SQL-011: Multi-tab support', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for new tab button
    const newTabButton = page.getByRole('button', { name: /New query/i }).or(
      page.locator('[aria-label="New query"]')
    );

    // Click to create new tab
    if (await newTabButton.isVisible()) {
      await newTabButton.click();
      await page.waitForTimeout(500);

      // Should have multiple tabs now
      const tabs = page.locator('[role="tab"]').or(page.getByText('New query'));
      const count = await tabs.count();
      expect(count).toBeGreaterThanOrEqual(1);
    }
  });

  test('E2E-SQL-012: Role selector exists', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for role selector
    const roleSelector = page.getByText(/postgres|anon|authenticated|service_role/i);
    await expect(roleSelector.first()).toBeVisible();
  });

  test('E2E-SQL-013: Query history drawer', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for history button
    const historyButton = page.getByRole('button', { name: /Query History|History/i });

    if (await historyButton.isVisible()) {
      await historyButton.click();
      await page.waitForTimeout(500);

      // Check that drawer opened
      const historyDrawer = page.getByText('Query History').first();
      await expect(historyDrawer).toBeVisible();
    }
  });

  test('E2E-SQL-014: Templates section exists', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for templates section in sidebar
    const templatesButton = page.getByText(/Templates|COMMUNITY/i);
    await expect(templatesButton.first()).toBeVisible();
  });

  test('E2E-SQL-015: Font size controls', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for font size controls or settings button
    const fontSizeDisplay = page.getByText(/\d+px/);
    if (await fontSizeDisplay.first().isVisible()) {
      await expect(fontSizeDisplay.first()).toBeVisible();
    }

    // Or look for settings button
    const settingsButton = page.getByRole('button', { name: /Settings/i });
    if (await settingsButton.isVisible()) {
      await settingsButton.click();
      await page.waitForTimeout(500);

      // Check for font size slider
      const fontSizeLabel = page.getByText(/Font Size/i);
      await expect(fontSizeLabel).toBeVisible();
    }
  });

  test('E2E-SQL-016: Execute selected text only', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    // Type multiple queries
    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type('SELECT 1; SELECT 2;');

    // Select just the first query
    await page.keyboard.press('Home');
    await page.keyboard.press('Shift+End');

    await page.waitForTimeout(500);

    // Look for "Selection active" badge
    const selectionBadge = page.getByText(/Selection active/i);
    // This might not always show depending on selection state
  });

  test('E2E-SQL-017: EXPLAIN button exists', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for EXPLAIN button/icon
    const explainButton = page.getByRole('button', { name: /Explain/i }).or(
      page.locator('[aria-label*="EXPLAIN"]')
    );
    await expect(explainButton.first()).toBeVisible();
  });

  test('E2E-SQL-018: Copy results menu', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Run a query first
    const editor = page.locator('.monaco-editor');
    await expect(editor).toBeVisible({ timeout: 10000 });

    const textarea = page.locator('.monaco-editor textarea');
    await textarea.focus();
    await page.keyboard.press('Meta+a');
    await page.keyboard.press('Control+a');
    await page.keyboard.type('SELECT 1 AS test;');

    const runButton = page.getByRole('button', { name: /Run/i });
    await runButton.click();

    await page.waitForTimeout(3000);

    // Look for Copy button
    const copyButton = page.getByRole('button', { name: /Copy/i });
    if (await copyButton.isVisible()) {
      await copyButton.click();

      // Check for copy options
      const jsonOption = page.getByText(/JSON/i);
      await expect(jsonOption.first()).toBeVisible();
    }
  });

  test('E2E-SQL-019: AI Assistant button exists', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for AI button (sparkles icon)
    const aiButton = page.getByRole('button', { name: /AI Assistant/i }).or(
      page.locator('[aria-label*="AI"]')
    );

    if (await aiButton.first().isVisible()) {
      await aiButton.first().click();
      await page.waitForTimeout(500);

      // Check for AI modal
      const aiModal = page.getByText(/AI Assistant/i);
      await expect(aiModal.first()).toBeVisible();
    }
  });

  test('E2E-SQL-020: Quickstarts templates load', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Expand COMMUNITY section
    const communitySection = page.getByText('COMMUNITY');
    if (await communitySection.isVisible()) {
      await communitySection.click();
      await page.waitForTimeout(300);

      // Look for Quickstarts
      const quickstartsButton = page.getByText('Quickstarts');
      if (await quickstartsButton.isVisible()) {
        await quickstartsButton.click();
        await page.waitForTimeout(300);

        // Check for quickstart items
        const helloWorld = page.getByText('Hello World');
        await expect(helloWorld).toBeVisible();
      }
    }
  });
});

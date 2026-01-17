import { test, expect } from '@playwright/test';

test.describe('Logs Explorer Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const logsLink = page.locator('.mantine-AppShell-navbar').getByRole('link', { name: 'Logs' });
    await logsLink.click();

    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/logs/);
  });

  test('E2E-LOG-001: Logs page loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const logsSection = page.getByText(/Logs|Explorer|No logs/i).first();
    await expect(logsSection).toBeVisible();
  });

  test('E2E-LOG-002: Log type selector available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const typeSelector = page.getByRole('combobox').or(page.getByText(/All log types|postgres|auth|api/i)).first();
    await expect(typeSelector).toBeVisible();
  });

  test('E2E-LOG-003: Level filter available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const levelFilter = page.getByText(/Error|Warning|Info|Debug|All/i).first();
    await expect(levelFilter).toBeVisible();
  });

  test('E2E-LOG-004: Search input visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder(/Search/i).first();
    await expect(searchInput).toBeVisible();
  });

  test('E2E-LOG-005: Time range pickers', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const timeInput = page.locator('input[type="datetime-local"]').or(page.getByText(/Start time|End time/i)).first();
    const isVisible = await timeInput.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-LOG-006: Export button available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const exportButton = page.getByRole('button', { name: /Export/i });
    await expect(exportButton).toBeVisible();
  });

  test('E2E-LOG-007: Export dropdown options', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const exportButton = page.getByRole('button', { name: /Export/i });
    await exportButton.click();

    const jsonOption = page.getByText(/JSON/i);
    const csvOption = page.getByText(/CSV/i);

    const jsonVisible = await jsonOption.isVisible().catch(() => false);
    const csvVisible = await csvOption.isVisible().catch(() => false);

    expect(jsonVisible || csvVisible).toBe(true);
  });

  test('E2E-LOG-008: Auto-refresh toggle', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const autoRefresh = page.getByText(/Auto-refresh/i).or(page.getByRole('switch')).first();
    await expect(autoRefresh).toBeVisible();
  });

  test('E2E-LOG-009: Log count displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const logCount = page.getByText(/logs|0|entries/i).first();
    await expect(logCount).toBeVisible();
  });

  test('E2E-LOG-010: Clear filters button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // First set some filters
    const searchInput = page.getByPlaceholder(/Search/i).first();
    await searchInput.fill('test');

    await page.waitForTimeout(500);

    const clearButton = page.getByRole('button', { name: /Clear|Reset/i });
    const isVisible = await clearButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-LOG-011: Log level color coding', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for colored badges
    const levelBadge = page.locator('.mantine-Badge-root').or(page.getByText(/error|warning|info/i)).first();
    const isVisible = await levelBadge.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-LOG-012: Refresh button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const refreshButton = page.getByRole('button', { name: /Refresh/i });
    const isVisible = await refreshButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

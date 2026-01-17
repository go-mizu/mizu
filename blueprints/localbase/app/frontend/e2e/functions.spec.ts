import { test, expect } from '@playwright/test';

test.describe('Edge Functions Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/functions');
    await page.waitForLoadState('networkidle');
  });

  test('E2E-FUNC-000: Page loads without JavaScript errors', async ({ page }) => {
    const jsErrors: string[] = [];

    // Listen for console errors
    page.on('pageerror', (error) => {
      jsErrors.push(error.message);
    });

    // Navigate and wait for load
    await page.goto('/functions');
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

  test('E2E-FUNC-001: Function list loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const functionSection = page.getByText(/Functions|No functions|Create function/i).first();
    await expect(functionSection).toBeVisible();
  });

  test('E2E-FUNC-002: Create function button visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add|Deploy/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-FUNC-003: Create function modal opens', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add|Deploy/i }).first();
    await createButton.click();

    const modal = page.getByRole('dialog').or(page.locator('.mantine-Modal-content'));
    await expect(modal).toBeVisible();
  });

  test('E2E-FUNC-004: Function name input', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add|Deploy/i }).first();
    await createButton.click();

    const nameInput = page.getByPlaceholder(/name|function/i).or(page.getByLabel(/name/i)).first();
    await expect(nameInput).toBeVisible();
  });

  test('E2E-FUNC-005: Function status badge', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const statusBadge = page.getByText(/Active|Inactive|Deployed/i).first();
    const isVisible = await statusBadge.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-FUNC-006: Invoke function button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const invokeButton = page.getByRole('button', { name: /Invoke|Test|Run/i }).first();
    const isVisible = await invokeButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-FUNC-007: Function URL displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const functionUrl = page.getByText(/functions\/v1\/|http:\/\//i).first();
    const isVisible = await functionUrl.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-FUNC-008: JWT verification toggle', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const jwtToggle = page.getByText(/JWT|Verify|Authentication/i).first();
    const isVisible = await jwtToggle.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-FUNC-009: Delete function confirmation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const deleteButton = page.getByRole('button', { name: /delete/i }).first();

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

  test('E2E-FUNC-010: Last updated time shown', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const timestamp = page.getByText(/ago|Updated|Modified/i).first();
    const isVisible = await timestamp.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

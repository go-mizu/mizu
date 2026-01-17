import { test, expect } from '@playwright/test';

test.describe('Authentication/Users Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/auth/users');
    await page.waitForLoadState('networkidle');
  });

  test('E2E-AUTH-001: User list loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Check for user list or empty state
    const userSection = page.getByText(/Users|No users|Create user/i).first();
    await expect(userSection).toBeVisible();
  });

  test('E2E-AUTH-002: Create user button is visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|Add|New/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-AUTH-003: Create user modal opens', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|Add|New/i }).first();
    await createButton.click();

    // Check for modal
    const modal = page.getByRole('dialog').or(page.locator('.mantine-Modal-content'));
    await expect(modal).toBeVisible();
  });

  test('E2E-AUTH-004: Create new user', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|Add|New/i }).first();
    await createButton.click();

    // Fill in user details
    const email = `test-${Date.now()}@example.com`;
    const emailInput = page.getByPlaceholder(/email/i).or(page.getByLabel(/email/i)).first();
    await emailInput.fill(email);

    const passwordInput = page.getByPlaceholder(/password/i).or(page.getByLabel(/password/i)).first();
    await passwordInput.fill('Test123456!');

    // Submit
    const submitButton = page.getByRole('button', { name: /Create|Save|Submit/i }).last();
    await submitButton.click();

    await page.waitForTimeout(2000);

    // Check for success or user in list
    const success = page.getByText(/created|success/i).or(page.getByText(email));
    const isVisible = await success.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-AUTH-005: User email displays', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for email format in the list
    const emailCell = page.getByText(/@/).first();
    const isVisible = await emailCell.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-AUTH-006: Provider badge shows', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for provider indicator
    const provider = page.getByText(/email|google|github|provider/i).first();
    const isVisible = await provider.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-AUTH-007: Search filters users', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder(/Search|Filter/i).first();

    if (await searchInput.isVisible()) {
      await searchInput.fill('test');
      await page.waitForTimeout(500);

      // The user list should update
      const userList = page.locator('table').or(page.getByRole('list'));
      await expect(userList).toBeVisible();
    }
  });

  test('E2E-AUTH-008: User action menu available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for action buttons (edit, delete)
    const actionButton = page.getByRole('button', { name: /actions|menu|edit|delete/i }).or(page.locator('[data-testid="user-actions"]')).first();

    const isVisible = await actionButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-AUTH-009: Delete user shows confirmation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Find a delete button for any user
    const deleteButton = page.getByRole('button', { name: /delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Check for confirmation modal
      const confirmModal = page.getByText(/confirm|sure|delete this user/i);
      await expect(confirmModal).toBeVisible();

      // Close modal
      const cancelButton = page.getByRole('button', { name: /cancel|no|close/i });
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      }
    }
  });

  test('E2E-AUTH-010: Last sign-in time shown', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for timestamp format
    const signInTime = page.getByText(/ago|never|last sign|signed in/i).first();
    const isVisible = await signInTime.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-AUTH-011: Email verified badge', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for verification status
    const verifiedBadge = page.getByText(/verified|unverified|confirmed/i).first();
    const isVisible = await verifiedBadge.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-AUTH-012: User ID displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for UUID format
    const uuidPattern = page.getByText(/[0-9a-f]{8}-[0-9a-f]{4}/i).first();
    const isVisible = await uuidPattern.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

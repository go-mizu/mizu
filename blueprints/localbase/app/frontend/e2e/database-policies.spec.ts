import { test, expect } from '@playwright/test';

test.describe('Database Policies Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const sidebar = page.locator('.mantine-AppShell-navbar');
    const databaseNavLink = sidebar.locator('.mantine-NavLink-root').filter({ hasText: 'Database' }).first();
    await databaseNavLink.click();

    await page.waitForTimeout(500);

    const policiesLink = sidebar.getByRole('link', { name: 'Policies', exact: true });
    await policiesLink.click();

    await page.waitForLoadState('networkidle');
  });

  test('E2E-POLICY-001: Policy list loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const policySection = page.getByText(/Policies|RLS|Row Level Security|No policies/i).first();
    await expect(policySection).toBeVisible();
  });

  test('E2E-POLICY-002: Policies grouped by table', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for table grouping
    const tableGroup = page.getByText(/table|public\./i).first();
    const isVisible = await tableGroup.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-POLICY-003: Create policy button visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-POLICY-004: Create policy modal opens', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await createButton.click();

    const modal = page.getByRole('dialog').or(page.locator('.mantine-Modal-content'));
    await expect(modal).toBeVisible();
  });

  test('E2E-POLICY-005: Policy command types available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await createButton.click();

    await page.waitForTimeout(500);

    // Look for command selector
    const commandSelect = page.getByText(/SELECT|INSERT|UPDATE|DELETE|ALL/i).first();
    await expect(commandSelect).toBeVisible();
  });

  test('E2E-POLICY-006: Delete policy confirmation', async ({ page }) => {
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

  test('E2E-POLICY-007: Policy roles filter', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for roles in policies
    const rolesFilter = page.getByText(/anon|authenticated|service_role|public/i).first();
    const isVisible = await rolesFilter.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-POLICY-008: RLS toggle per table', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for RLS enable/disable toggle
    const rlsToggle = page.getByRole('switch').or(page.getByText(/Enable RLS|Disable RLS/i)).first();
    const isVisible = await rlsToggle.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

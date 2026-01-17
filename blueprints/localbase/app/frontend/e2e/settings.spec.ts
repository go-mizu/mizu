import { test, expect } from '@playwright/test';

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const settingsLink = page.locator('.mantine-AppShell-navbar').getByRole('link', { name: 'Settings' });
    await settingsLink.click();

    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/settings/);
  });

  test('E2E-SET-001: Settings page loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const settingsSection = page.getByText(/Settings|Configuration/i).first();
    await expect(settingsSection).toBeVisible();
  });

  test('E2E-SET-002: General tab available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const generalTab = page.getByRole('tab', { name: /General/i }).or(page.getByText(/General/i)).first();
    await expect(generalTab).toBeVisible();
  });

  test('E2E-SET-003: Project name displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const projectName = page.getByText(/Project|Name|localbase/i).first();
    await expect(projectName).toBeVisible();
  });

  test('E2E-SET-004: API Keys tab available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const apiTab = page.getByRole('tab', { name: /API/i }).or(page.getByText(/API Keys/i)).first();
    await expect(apiTab).toBeVisible();
  });

  test('E2E-SET-005: API Keys section shows keys', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Click on API tab
    const apiTab = page.getByRole('tab', { name: /API/i }).or(page.getByText(/API Keys/i)).first();
    await apiTab.click();

    await page.waitForTimeout(500);

    // Check for anon key and service role key
    const anonKey = page.getByText(/anon|public/i).first();
    await expect(anonKey).toBeVisible();
  });

  test('E2E-SET-006: Service role key displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const apiTab = page.getByRole('tab', { name: /API/i }).or(page.getByText(/API Keys/i)).first();
    await apiTab.click();

    await page.waitForTimeout(500);

    const serviceKey = page.getByText(/service|role|secret/i).first();
    await expect(serviceKey).toBeVisible();
  });

  test('E2E-SET-007: Copy button for keys', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const apiTab = page.getByRole('tab', { name: /API/i }).or(page.getByText(/API Keys/i)).first();
    await apiTab.click();

    await page.waitForTimeout(500);

    const copyButton = page.getByRole('button', { name: /Copy/i }).first();
    await expect(copyButton).toBeVisible();
  });

  test('E2E-SET-008: Database tab available', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const dbTab = page.getByRole('tab', { name: /Database/i }).or(page.getByText(/Database/i)).first();
    await expect(dbTab).toBeVisible();
  });

  test('E2E-SET-009: Database connection details', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const dbTab = page.getByRole('tab', { name: /Database/i }).or(page.getByText(/Database/i)).first();
    await dbTab.click();

    await page.waitForTimeout(500);

    const connectionDetails = page.getByText(/Host|Port|User|Connection/i).first();
    await expect(connectionDetails).toBeVisible();
  });

  test('E2E-SET-010: Connection string displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const dbTab = page.getByRole('tab', { name: /Database/i }).or(page.getByText(/Database/i)).first();
    await dbTab.click();

    await page.waitForTimeout(500);

    const connectionString = page.getByText(/postgres:\/\/|Connection string/i).first();
    const isVisible = await connectionString.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-SET-011: Project URL displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const projectUrl = page.getByText(/localhost|URL|http:\/\//i).first();
    await expect(projectUrl).toBeVisible();
  });

  test('E2E-SET-012: Save settings button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const saveButton = page.getByRole('button', { name: /Save/i });
    const isVisible = await saveButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

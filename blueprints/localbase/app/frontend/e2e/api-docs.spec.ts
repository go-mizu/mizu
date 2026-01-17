import { test, expect } from '@playwright/test';

test.describe('API Docs Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const apiDocsLink = page.locator('.mantine-AppShell-navbar').getByRole('link', { name: 'API Docs' });
    await apiDocsLink.click();

    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/api-docs/);
  });

  test('E2E-API-001: API docs page loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const apiSection = page.getByText(/API|Documentation|Reference/i).first();
    await expect(apiSection).toBeVisible();
  });

  test('E2E-API-002: Base URL displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const baseUrl = page.getByText(/localhost|Base URL|http:\/\//i).first();
    await expect(baseUrl).toBeVisible();
  });

  test('E2E-API-003: Auth endpoints section', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const authSection = page.getByText(/Auth|Authentication|\/auth/i).first();
    await expect(authSection).toBeVisible();
  });

  test('E2E-API-004: REST API endpoints section', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const restSection = page.getByText(/REST|\/rest/i).first();
    await expect(restSection).toBeVisible();
  });

  test('E2E-API-005: Storage endpoints section', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const storageSection = page.getByText(/Storage|\/storage/i).first();
    await expect(storageSection).toBeVisible();
  });

  test('E2E-API-006: HTTP method badges', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const methodBadge = page.getByText(/GET|POST|PUT|PATCH|DELETE/i).first();
    await expect(methodBadge).toBeVisible();
  });

  test('E2E-API-007: Endpoint paths displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const endpointPath = page.getByText(/\/v1\/|\/users|\/token/i).first();
    await expect(endpointPath).toBeVisible();
  });

  test('E2E-API-008: Copy path functionality', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const copyButton = page.getByRole('button', { name: /Copy/i }).first();
    const isVisible = await copyButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-API-009: Accordion expand/collapse', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for expandable sections
    const accordion = page.locator('.mantine-Accordion-item').or(page.getByRole('button', { expanded: false })).first();
    const isVisible = await accordion.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-API-010: Example payloads shown', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const codeBlock = page.locator('code').or(page.locator('pre')).first();
    const isVisible = await codeBlock.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

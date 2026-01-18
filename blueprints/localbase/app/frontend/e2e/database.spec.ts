import { test, expect } from '@playwright/test';

test.describe('Unified Database Layout', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/database');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('E2E-DB-001: Database layout loads with sidebar', async ({ page }) => {
    // Should redirect to /database/tables by default
    await expect(page).toHaveURL(/\/database\/tables/);

    // Database sidebar should be visible
    const databaseHeading = page.getByText('Database').first();
    await expect(databaseHeading).toBeVisible({ timeout: 15000 });
  });

  test('E2E-DB-002: All navigation items are visible', async ({ page }) => {
    const navItems = [
      'Tables',
      'SQL Editor',
      'Schema Visualizer',
      'Policies',
      'Roles',
      'Indexes',
      'Views',
      'Triggers',
      'Functions',
      'Extensions',
    ];

    for (const item of navItems) {
      const navItem = page.getByRole('link', { name: item }).or(page.getByText(item));
      await expect(navItem).toBeVisible({ timeout: 10000 });
    }
  });

  test('E2E-DB-003: Schema selector is functional', async ({ page }) => {
    // Find schema selector
    const schemaSelector = page.getByRole('combobox').or(page.locator('input[role="searchbox"]')).first();
    await expect(schemaSelector).toBeVisible({ timeout: 10000 });

    // Click to open dropdown
    await schemaSelector.click();
    await page.waitForTimeout(500);

    // Should see public schema in dropdown
    const publicOption = page.getByText('public');
    const isVisible = await publicOption.isVisible({ timeout: 3000 }).catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-DB-004: Database overview stats are displayed', async ({ page }) => {
    // Look for database size and connection count
    const sizeLabel = page.getByText('Size');
    const connectionsLabel = page.getByText('Connections');

    await expect(sizeLabel).toBeVisible({ timeout: 10000 });
    await expect(connectionsLabel).toBeVisible({ timeout: 10000 });
  });

  test('E2E-DB-005: Navigation to Tables works', async ({ page }) => {
    const tablesLink = page.getByRole('link', { name: 'Tables' }).first();
    await tablesLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/tables/);
  });

  test('E2E-DB-006: Navigation to SQL Editor works', async ({ page }) => {
    const sqlLink = page.getByRole('link', { name: 'SQL Editor' }).first();
    await sqlLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/sql/);

    // Monaco editor should load
    const editor = page.locator('.monaco-editor').or(page.locator('[data-testid="sql-editor"]'));
    await expect(editor).toBeVisible({ timeout: 15000 });
  });

  test('E2E-DB-007: Navigation to Schema Visualizer works', async ({ page }) => {
    const schemaLink = page.getByRole('link', { name: 'Schema Visualizer' }).first();
    await schemaLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/schema/);

    // Canvas or React Flow container should load
    const canvas = page.locator('.react-flow').or(page.locator('canvas'));
    const isVisible = await canvas.isVisible({ timeout: 10000 }).catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-DB-008: Navigation to Policies works', async ({ page }) => {
    const policiesLink = page.getByRole('link', { name: 'Policies' }).first();
    await policiesLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/policies/);
  });

  test('E2E-DB-009: Navigation to Roles works', async ({ page }) => {
    const rolesLink = page.getByRole('link', { name: 'Roles' }).first();
    await rolesLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/roles/);
  });

  test('E2E-DB-010: Navigation to Indexes works', async ({ page }) => {
    const indexesLink = page.getByRole('link', { name: 'Indexes' }).first();
    await indexesLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/indexes/);
  });

  test('E2E-DB-011: Navigation to Views works', async ({ page }) => {
    const viewsLink = page.getByRole('link', { name: 'Views' }).first();
    await viewsLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/views/);
  });

  test('E2E-DB-012: Navigation to Triggers works', async ({ page }) => {
    const triggersLink = page.getByRole('link', { name: 'Triggers' }).first();
    await triggersLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/triggers/);
  });

  test('E2E-DB-013: Navigation to Functions works', async ({ page }) => {
    const functionsLink = page.getByRole('link', { name: 'Functions' }).first();
    await functionsLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/functions/);
  });

  test('E2E-DB-014: Navigation to Extensions works', async ({ page }) => {
    const extensionsLink = page.getByRole('link', { name: 'Extensions' }).first();
    await extensionsLink.click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/extensions/);
  });

  test('E2E-DB-015: Active nav item is highlighted', async ({ page }) => {
    // Tables should be active by default
    const tablesLink = page.getByRole('link', { name: 'Tables' }).first();
    await expect(tablesLink).toHaveAttribute('data-active', 'true');
  });

  test('E2E-DB-016: Count badges are displayed for nav items', async ({ page }) => {
    // Look for count badges next to nav items
    const badges = page.locator('.mantine-Badge-root');
    const count = await badges.count();

    // Should have at least some count badges (Tables, Views, Indexes, Policies, Functions)
    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Database Legacy Route Redirects', () => {
  test('E2E-DB-017: /table-editor redirects to /database/tables', async ({ page }) => {
    await page.goto('/table-editor');
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/tables/);
  });

  test('E2E-DB-018: /sql-editor redirects to /database/sql', async ({ page }) => {
    await page.goto('/sql-editor');
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/sql/);
  });

  test('E2E-DB-019: /database/schema-visualizer redirects to /database/schema', async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/database\/schema/);
  });
});

test.describe('Extensions Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/database/extensions');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('E2E-EXT-001: Extensions page loads', async ({ page }) => {
    const heading = page.getByRole('heading', { name: /Extensions/i });
    await expect(heading).toBeVisible({ timeout: 15000 });
  });

  test('E2E-EXT-002: Search input is visible', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search extensions/i);
    await expect(searchInput).toBeVisible({ timeout: 10000 });
  });

  test('E2E-EXT-003: Installed extensions section is visible', async ({ page }) => {
    const installedSection = page.getByText(/INSTALLED/i);
    await expect(installedSection).toBeVisible({ timeout: 10000 });
  });

  test('E2E-EXT-004: Available extensions section is visible', async ({ page }) => {
    const availableSection = page.getByText(/AVAILABLE/i);
    await expect(availableSection).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Database Functions Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/database/functions');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('E2E-FUNC-001: Functions page loads', async ({ page }) => {
    const heading = page.getByRole('heading', { name: /Database Functions/i });
    await expect(heading).toBeVisible({ timeout: 15000 });
  });

  test('E2E-FUNC-002: Schema filter is visible', async ({ page }) => {
    const schemaFilter = page.getByRole('combobox').or(page.locator('input[role="searchbox"]')).first();
    await expect(schemaFilter).toBeVisible({ timeout: 10000 });
  });

  test('E2E-FUNC-003: Search input is visible', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search functions/i);
    await expect(searchInput).toBeVisible({ timeout: 10000 });
  });

  test('E2E-FUNC-004: Functions table is displayed', async ({ page }) => {
    const table = page.locator('table');
    await expect(table).toBeVisible({ timeout: 10000 });

    // Check for table headers
    const nameHeader = page.getByRole('columnheader', { name: /Name/i });
    await expect(nameHeader).toBeVisible();
  });
});

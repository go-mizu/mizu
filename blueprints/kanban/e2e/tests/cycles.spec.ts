import { test, expect, testUsers } from '../fixtures/test-fixtures.js';
import { ApiHelper } from '../helpers/api.js';

test.describe('Cycles Management', () => {
  test.describe('Cycles Page', () => {
    test('TC-CYCLE-001: cycles page shows active cycle', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      // Should show active cycle section
      await expect(page.locator('h2:has-text("Active")')).toBeVisible();

      // Should have an active cycle card
      const activeCycle = page.locator('.card:has(.status-badge:has-text("Active"))');
      await expect(activeCycle).toBeVisible();
    });

    test('TC-CYCLE-002: cycles page shows upcoming cycles', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      // Should show upcoming section
      await expect(page.locator('h2:has-text("Upcoming")')).toBeVisible();
    });

    test('TC-CYCLE-003: cycles page shows completed cycles', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      // Should show completed section
      await expect(page.locator('h2:has-text("Completed")')).toBeVisible();
    });

    test('TC-CYCLE-004: active cycle shows progress bar', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      const activeCycle = page.locator('.card:has(.status-badge:has-text("Active"))');

      // Should have progress bar
      const progressBar = activeCycle.locator('.h-2.bg-tertiary');
      await expect(progressBar).toBeVisible();
    });

    test('TC-CYCLE-005: active cycle shows issue count', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      const activeCycle = page.locator('.card:has(.status-badge:has-text("Active"))');

      // Should show issue count
      await expect(activeCycle.locator('text=/\\d+ issues/')).toBeVisible();
    });
  });

  test.describe('Create Cycle', () => {
    test('TC-CYCLE-006: can open create cycle modal', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      // Click create button
      await page.locator('button:has-text("New Cycle")').click();

      // Modal should open
      await expect(page.locator('#create-cycle-modal')).toBeVisible();
    });

    test('TC-CYCLE-007: create cycle modal has required fields', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');
      await page.locator('button:has-text("New Cycle")').click();

      const modal = page.locator('#create-cycle-modal');

      // Should have name, start date, end date
      await expect(modal.locator('#cycle-name')).toBeVisible();
      await expect(modal.locator('#cycle-start')).toBeVisible();
      await expect(modal.locator('#cycle-end')).toBeVisible();
    });
  });

  test.describe('Cycle Navigation', () => {
    test('TC-CYCLE-008: clicking cycle card goes to cycle detail', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      // Click on an upcoming or completed cycle
      const cycleCard = page.locator('.card:has(.status-badge:has-text("Planning"))').first();

      if (await cycleCard.count() > 0) {
        await cycleCard.click();

        // Should navigate to cycle detail
        await expect(page).toHaveURL(/cycle/);
      }
    });

    test('TC-CYCLE-009: active cycle has view issues button', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      const activeCycle = page.locator('.card:has(.status-badge:has-text("Active"))');

      // Should have view issues button
      await expect(activeCycle.locator('a:has-text("View Issues")')).toBeVisible();
    });
  });

  test.describe('Cycle Status', () => {
    test('TC-CYCLE-010: cycle shows correct status badges', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      // Should have various status badges
      await expect(page.locator('.status-badge:has-text("Active")')).toBeVisible();

      // Should have planning or completed badges
      const planningOrCompleted = page.locator('.status-badge:has-text("Planning"), .status-badge:has-text("Completed")');
      await expect(planningOrCompleted.first()).toBeVisible();
    });

    test('TC-CYCLE-011: completed cycles show completion count', async ({ page, loginAs }) => {
      await loginAs('alice');

      await page.goto('/w/acme/cycles');

      const completedSection = page.locator('h2:has-text("Completed")').locator('..');
      const completedCycle = completedSection.locator('.card').first();

      if (await completedCycle.count() > 0) {
        // Should show completed count like "2 / 3 completed"
        await expect(completedCycle.locator('text=/\\d+ \\/ \\d+ completed/')).toBeVisible();
      }
    });
  });
});

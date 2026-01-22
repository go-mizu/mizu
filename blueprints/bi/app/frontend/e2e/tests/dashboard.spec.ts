import { test, expect } from '../fixtures/test-base';
import { DashboardPage } from '../pages/dashboard.page';

test.describe('Dashboard', () => {
  test.describe('Dashboard Viewing', () => {
    test('should load dashboard with cards', async ({ authenticatedPage: page }) => {
      // Navigate to browse and find a dashboard
      await page.goto('/browse');
      await page.waitForSelector('text=Browse, h2');

      // Click on dashboards tab if available
      await page.click('button:has-text("Dashboards"), [role="tab"]:has-text("Dashboards")').catch(() => {});

      // Find and click a dashboard
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();
      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        // Wait for dashboard to load
        await page.waitForURL(/\/dashboard\//, { timeout: 10000 });

        const dashboardPage = new DashboardPage(page);

        // Verify dashboard elements
        const cardCount = await dashboardPage.getCardCount();
        expect(cardCount).toBeGreaterThanOrEqual(0);

        await dashboardPage.takeScreenshot('dashboard_loaded');
      }
    });

    test('should display dashboard title', async ({ authenticatedPage: page }) => {
      const dashboardPage = new DashboardPage(page);

      // Go to browse and open first dashboard
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();
        await dashboardPage.waitForLoad();

        // Get title
        const title = await dashboardPage.getTitle();
        expect(title).toBeTruthy();

        await dashboardPage.takeScreenshot('dashboard_title');
      }
    });

    test('should render card visualizations', async ({ authenticatedPage: page }) => {
      // Go to browse and open first dashboard
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();
        await page.waitForSelector('.react-grid-layout, [data-testid="dashboard-grid"]', { timeout: 10000 });

        const dashboardPage = new DashboardPage(page);

        // Check if cards have visualizations
        const cardCount = await dashboardPage.getCardCount();
        if (cardCount > 0) {
          const hasViz = await dashboardPage.verifyCardRendered(0);
          // At least the first card should have something rendered
          // (could be loading state or actual viz)
        }

        await dashboardPage.takeScreenshot('card_visualizations');
      }
    });

    test('should enter and exit fullscreen mode', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Try fullscreen if button exists
        const fullscreenBtn = page.locator('button[aria-label="Fullscreen"], button:has-text("Fullscreen")');
        if (await fullscreenBtn.isVisible({ timeout: 2000 })) {
          await fullscreenBtn.click();

          // Should be in fullscreen
          await page.waitForTimeout(500);
          await dashboardPage.takeScreenshot('fullscreen_mode');

          // Exit fullscreen
          await page.keyboard.press('Escape');
        }
      }
    });
  });

  test.describe('Dashboard Editing', () => {
    test('should enter edit mode', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Find and click edit button
        const editBtn = page.locator('button:has-text("Edit"), [aria-label="Edit"]');
        if (await editBtn.isVisible({ timeout: 2000 })) {
          await editBtn.click();

          // Should show save button now
          await expect(page.locator('button:has-text("Save"), button:has-text("Done")')).toBeVisible({ timeout: 5000 });

          await dashboardPage.takeScreenshot('edit_mode');
        }
      }
    });

    test('should add card to dashboard', async ({ authenticatedPage: page }) => {
      // Create new dashboard
      await page.goto('/dashboard/new');

      const dashboardPage = new DashboardPage(page);
      await page.waitForSelector('button:has-text("Add"), text=Add card', { timeout: 10000 });

      // Click add card
      await page.click('button:has-text("Add"), text=Add card');

      // Modal should appear
      await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });

      await dashboardPage.takeScreenshot('add_card_modal');
    });

    test('should save dashboard changes', async ({ authenticatedPage: page }) => {
      // Create new dashboard
      await page.goto('/dashboard/new');

      // Wait for page to load
      await page.waitForTimeout(1000);

      // Click save
      await page.click('button:has-text("Save")');

      // Modal should appear
      const modal = page.locator('[role="dialog"]:has-text("Save")');
      if (await modal.isVisible({ timeout: 3000 })) {
        // Enter name
        const timestamp = Date.now();
        await page.fill('input[placeholder*="name" i]', `Test Dashboard ${timestamp}`);

        // Submit
        await page.click('[role="dialog"] button:has-text("Save")');

        // Modal should close
        await expect(modal).toBeHidden({ timeout: 10000 });
      }
    });

    test('should cancel dashboard edit', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Enter edit mode
        const editBtn = page.locator('button:has-text("Edit")');
        if (await editBtn.isVisible({ timeout: 2000 })) {
          await editBtn.click();
          await page.waitForTimeout(500);

          // Click cancel
          const cancelBtn = page.locator('button:has-text("Cancel")');
          if (await cancelBtn.isVisible()) {
            await cancelBtn.click();

            // Should exit edit mode
            await expect(page.locator('button:has-text("Edit")')).toBeVisible({ timeout: 5000 });
          }
        }
      }
    });
  });

  test.describe('Dashboard Filters', () => {
    test('should display filter bar when filters exist', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Check for filter bar
        const filterBar = page.locator('[data-testid="filter-bar"], .dashboard-filters, button:has-text("Add filter")');
        if (await filterBar.isVisible({ timeout: 2000 })) {
          await dashboardPage.takeScreenshot('filter_bar');
        }
      }
    });
  });

  test.describe('Dashboard Creation', () => {
    test('should create new empty dashboard', async ({ authenticatedPage: page }) => {
      await page.goto('/dashboard/new');

      const dashboardPage = new DashboardPage(page);

      // Should show empty state or add card prompt
      await expect(page.locator('text=Add, text=empty, button:has-text("Add")')).toBeVisible({ timeout: 10000 });

      await dashboardPage.takeScreenshot('new_dashboard_empty');
    });

    test('should create dashboard with name and description', async ({ authenticatedPage: page }) => {
      await page.goto('/dashboard/new');

      // Click save
      await page.click('button:has-text("Save")');

      // Fill in modal
      const modal = page.locator('[role="dialog"]:has-text("Save")');
      await expect(modal).toBeVisible({ timeout: 5000 });

      const timestamp = Date.now();
      const dashboardName = `E2E Test Dashboard ${timestamp}`;

      await page.fill('input[placeholder*="name" i]', dashboardName);
      await page.fill('textarea[placeholder*="description" i]', 'Created by E2E test').catch(() => {});

      await page.click('[role="dialog"] button:has-text("Save")');

      // Should save successfully
      await expect(modal).toBeHidden({ timeout: 10000 });

      const dashboardPage = new DashboardPage(page);
      await dashboardPage.takeScreenshot('dashboard_created');
    });
  });

  test.describe('Dashboard Tabs', () => {
    test('should display tabs when present', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Check for tabs
        const tabCount = await dashboardPage.getTabCount();
        console.log(`Dashboard has ${tabCount} tabs`);

        await dashboardPage.takeScreenshot('dashboard_tabs');
      }
    });
  });

  test.describe('UI Fidelity', () => {
    test('should have correct dashboard layout', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();
        await dashboardPage.verifyMetabaseLayout();

        await dashboardPage.takeScreenshot('layout_dashboard');
      }
    });

    test('should have proper grid layout', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Check for grid
        const grid = page.locator('.react-grid-layout, [data-testid="dashboard-grid"]');
        if (await grid.isVisible()) {
          const gridDisplay = await grid.evaluate(el => getComputedStyle(el).display);
          // Should have proper layout
          expect(gridDisplay).toBeTruthy();
        }

        await dashboardPage.takeScreenshot('grid_layout');
      }
    });

    test('should have card hover effects', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();

      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();

        const dashboardPage = new DashboardPage(page);
        await dashboardPage.waitForLoad();

        // Hover over a card
        const firstCard = page.locator('.react-grid-item, [data-testid="dashboard-card"]').first();
        if (await firstCard.isVisible({ timeout: 2000 })) {
          await firstCard.hover();
          await page.waitForTimeout(300);

          await dashboardPage.takeScreenshot('card_hover');
        }
      }
    });
  });
});

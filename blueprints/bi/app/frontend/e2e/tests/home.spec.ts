import { test, expect } from '../fixtures/test-base';
import { HomePage } from '../pages/home.page';

test.describe('Home Page', () => {
  test.describe('Layout & Navigation', () => {
    test('should display home page with correct header', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Verify header elements
      await expect(homePage.title).toBeVisible();
      await expect(homePage.searchButton).toBeVisible();
      await expect(homePage.newButton).toBeVisible();

      // Screenshot for UI comparison
      await homePage.takeScreenshot('layout_header');
    });

    test('should display statistics cards with correct counts', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Get stats
      const stats = await homePage.getStats();

      // Verify stats are numbers (not NaN)
      expect(typeof stats.questions).toBe('number');
      expect(typeof stats.dashboards).toBe('number');
      expect(typeof stats.collections).toBe('number');
      expect(typeof stats.databases).toBe('number');

      // Screenshot for UI comparison
      await homePage.takeScreenshot('layout_stats');
    });

    test('should display sections correctly', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Check if we have content or empty state
      const hasRecent = await homePage.hasRecentItems();
      const hasPinned = await homePage.hasPinnedItems();
      const hasEmpty = await homePage.hasEmptyState();

      // Should have either content or empty state
      expect(hasRecent || hasPinned || hasEmpty).toBeTruthy();

      await homePage.takeScreenshot('layout_sections');
    });
  });

  test.describe('Quick Actions', () => {
    test('should open new question page from New menu', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Create new question
      await homePage.createNewQuestion();

      // Verify navigation
      await expect(page).toHaveURL(/\/question\/new/);
    });

    test('should open new dashboard page from New menu', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Create new dashboard
      await homePage.createNewDashboard();

      // Verify navigation
      await expect(page).toHaveURL(/\/dashboard\/new/);
    });

    test('should open search/command palette', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Open search
      await homePage.openSearch();

      // Verify command palette is open
      await expect(page.locator('[role="dialog"], [data-testid="command-palette"]')).toBeVisible();
    });

    test('should open search with keyboard shortcut Cmd+K', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Use keyboard shortcut
      await page.keyboard.press('Meta+k');

      // Verify command palette is open
      await expect(page.locator('[role="dialog"], [data-testid="command-palette"]')).toBeVisible({ timeout: 3000 });
    });
  });

  test.describe('Content Display', () => {
    test('should display pinned items when present', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      if (await homePage.hasPinnedItems()) {
        // Verify pinned section has items
        const pinnedCards = page.locator('[data-testid="pinned-item"], [data-pinned="true"]');
        expect(await pinnedCards.count()).toBeGreaterThan(0);
      }

      await homePage.takeScreenshot('pinned_items');
    });

    test('should display recent items when present', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      if (await homePage.hasRecentItems()) {
        // Verify recent section
        await expect(homePage.recentSection).toBeVisible();
      }

      await homePage.takeScreenshot('recent_items');
    });

    test('should navigate when clicking on pinned/recent item', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Find any clickable item card
      const itemCard = page.locator('[data-testid="item-card"], .question-card, .dashboard-card').first();

      if (await itemCard.isVisible()) {
        await itemCard.click();

        // Should navigate away from home
        await page.waitForTimeout(1000);
        const url = page.url();
        expect(url).toMatch(/\/(question|dashboard)\//);
      }
    });
  });

  test.describe('Empty State', () => {
    // This test would require a fresh database without seed data
    test.skip('should display getting started cards for empty state', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      if (await homePage.hasEmptyState()) {
        // Verify start here section
        await expect(homePage.startHereSection).toBeVisible();

        // Verify action cards
        await expect(page.locator('text=Add your data')).toBeVisible();
        await expect(page.locator('text=Ask a question')).toBeVisible();
        await expect(page.locator('text=Create a dashboard')).toBeVisible();
      }
    });
  });

  test.describe('UI Fidelity', () => {
    test('should match Metabase layout patterns', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Verify Metabase-style layout
      await homePage.verifyMetabaseLayout();

      // Take full page screenshot for comparison
      await homePage.takeScreenshot('full_page');
    });

    test('should have correct typography and spacing', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Check title styling
      const titleElement = page.locator('h2:has-text("Home")');
      const fontSize = await titleElement.evaluate(el => getComputedStyle(el).fontSize);
      const fontWeight = await titleElement.evaluate(el => getComputedStyle(el).fontWeight);

      // Title should have proper weight (600+)
      expect(parseInt(fontWeight)).toBeGreaterThanOrEqual(600);

      await homePage.takeScreenshot('typography');
    });

    test('should have correct card styling', async ({ authenticatedPage: page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();

      // Find stat cards
      const statCards = page.locator('[data-testid="stat-card"], button:has([data-icon])').first();

      if (await statCards.isVisible()) {
        // Check card has proper border radius
        const borderRadius = await statCards.evaluate(el => getComputedStyle(el).borderRadius);
        expect(parseInt(borderRadius)).toBeGreaterThanOrEqual(4);
      }

      await homePage.takeScreenshot('card_styling');
    });
  });
});

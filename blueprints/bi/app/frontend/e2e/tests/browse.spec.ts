import { test, expect } from '../fixtures/test-base';

test.describe('Browse & Collections', () => {
  test.describe('Browse Page', () => {
    test('should display browse page with tabs', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Verify tabs exist
      const tabs = page.locator('[role="tab"], button:has-text("Questions"), button:has-text("Dashboards")');
      expect(await tabs.count()).toBeGreaterThanOrEqual(2);

      await page.screenshot({ path: 'e2e/screenshots/browse_page.png', fullPage: true });
    });

    test('should filter by Questions tab', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Click Questions tab
      await page.click('[role="tab"]:has-text("Questions")');
      await page.waitForTimeout(500);

      // Tab should be selected
      const questionsTab = page.locator('[role="tab"]:has-text("Questions")');
      await expect(questionsTab).toHaveAttribute('aria-selected', 'true');

      // Count is shown in tab (Questions (N))
      const tabText = await questionsTab.textContent();
      expect(tabText).toContain('Questions');

      await page.screenshot({ path: 'e2e/screenshots/browse_questions.png', fullPage: true });
    });

    test('should filter by Dashboards tab', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Click Dashboards tab
      await page.click('[role="tab"]:has-text("Dashboards")');
      await page.waitForTimeout(500);

      // Tab should be selected
      const dashboardsTab = page.locator('[role="tab"]:has-text("Dashboards")');
      await expect(dashboardsTab).toHaveAttribute('aria-selected', 'true');

      // Count is shown in tab (Dashboards (N))
      const tabText = await dashboardsTab.textContent();
      expect(tabText).toContain('Dashboards');

      await page.screenshot({ path: 'e2e/screenshots/browse_dashboards.png', fullPage: true });
    });

    test('should search/filter items', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Look for search input
      const searchInput = page.locator('input[placeholder*="Search" i]');
      await expect(searchInput).toBeVisible({ timeout: 5000 });

      // Type in search
      await searchInput.fill('test');
      await page.waitForTimeout(500);

      // Results should update (page doesn't error)
      await page.screenshot({ path: 'e2e/screenshots/browse_search.png', fullPage: true });
    });

    test('should sort items', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Look for sort control
      const sortControl = page.locator('button:has-text("Sort"), [data-testid="sort-control"]');
      if (await sortControl.isVisible({ timeout: 3000 })) {
        await sortControl.click();

        // Verify sort options appear
        await expect(page.locator('[role="option"], [role="menuitem"]')).toBeVisible({ timeout: 3000 });

        await page.screenshot({ path: 'e2e/screenshots/browse_sort.png', fullPage: true });
      }
    });
  });

  test.describe('Collections', () => {
    test('should display collections tab', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Click Collections tab if available
      const collectionsTab = page.locator('button:has-text("Collections"), [role="tab"]:has-text("Collections")');
      if (await collectionsTab.isVisible({ timeout: 3000 })) {
        await collectionsTab.click();
        await page.waitForTimeout(500);

        await page.screenshot({ path: 'e2e/screenshots/browse_collections.png', fullPage: true });
      }
    });

    test('should navigate to collection page', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');

      // Click on collections tab
      await page.click('button:has-text("Collections")').catch(() => {});
      await page.waitForTimeout(500);

      // Find a collection card
      const collectionCard = page.locator('[data-type="collection"], .collection-card').first();
      if (await collectionCard.isVisible({ timeout: 3000 })) {
        await collectionCard.click();

        // Should navigate to collection (route is /browse/:id)
        await page.waitForURL(/\/browse\//, { timeout: 10000 });

        await page.screenshot({ path: 'e2e/screenshots/collection_page.png', fullPage: true });
      }
    });

    test('should create new collection', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');

      // Look for new collection button
      const newCollectionBtn = page.locator('button:has-text("New collection"), button:has-text("Create collection")');
      if (await newCollectionBtn.isVisible({ timeout: 3000 })) {
        await newCollectionBtn.click();

        // Modal should appear
        await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });

        // Enter name
        const timestamp = Date.now();
        await page.fill('input[placeholder*="name" i]', `Test Collection ${timestamp}`);

        await page.screenshot({ path: 'e2e/screenshots/new_collection_modal.png', fullPage: true });

        // Cancel for cleanup
        await page.click('button:has-text("Cancel")').catch(() => page.keyboard.press('Escape'));
      }
    });

    test('should display collection contents', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Collections")').catch(() => {});
      await page.waitForTimeout(500);

      const collectionCard = page.locator('[data-type="collection"], .collection-card').first();
      if (await collectionCard.isVisible({ timeout: 3000 })) {
        await collectionCard.click();
        await page.waitForURL(/\/browse\//, { timeout: 10000 });

        // Verify collection contents area
        await expect(page.locator('[data-testid="collection-items"], .collection-contents, main')).toBeVisible();

        await page.screenshot({ path: 'e2e/screenshots/collection_contents.png', fullPage: true });
      }
    });

    test('should move item to collection', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Find an item with menu
      const itemCard = page.locator('[data-type="question"], [data-type="dashboard"]').first();
      if (await itemCard.isVisible({ timeout: 3000 })) {
        // Hover to show menu
        await itemCard.hover();

        // Look for more actions menu
        const menuBtn = itemCard.locator('button[aria-label="More"], button:has([data-icon="dots"])');
        if (await menuBtn.isVisible({ timeout: 2000 })) {
          await menuBtn.click();

          // Look for move option
          const moveOption = page.locator('[role="menuitem"]:has-text("Move")');
          if (await moveOption.isVisible({ timeout: 2000 })) {
            await page.screenshot({ path: 'e2e/screenshots/item_move_menu.png', fullPage: true });
          }
        }
      }
    });
  });

  test.describe('Item Actions', () => {
    test('should open item details/info', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      const itemCard = page.locator('[data-type="question"], [data-type="dashboard"]').first();
      if (await itemCard.isVisible({ timeout: 3000 })) {
        await itemCard.hover();

        const menuBtn = itemCard.locator('button[aria-label="More"]');
        if (await menuBtn.isVisible({ timeout: 2000 })) {
          await menuBtn.click();

          const infoOption = page.locator('[role="menuitem"]:has-text("Info"), [role="menuitem"]:has-text("Details")');
          if (await infoOption.isVisible({ timeout: 2000 })) {
            await infoOption.click();

            // Info panel should appear
            await expect(page.locator('[data-testid="info-panel"], [role="dialog"]:has-text("Info")')).toBeVisible({ timeout: 5000 });

            await page.screenshot({ path: 'e2e/screenshots/item_info.png', fullPage: true });
          }
        }
      }
    });

    test('should duplicate item', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      const itemCard = page.locator('[data-type="question"]').first();
      if (await itemCard.isVisible({ timeout: 3000 })) {
        await itemCard.hover();

        const menuBtn = itemCard.locator('button[aria-label="More"]');
        if (await menuBtn.isVisible({ timeout: 2000 })) {
          await menuBtn.click();

          const duplicateOption = page.locator('[role="menuitem"]:has-text("Duplicate")');
          if (await duplicateOption.isVisible({ timeout: 2000 })) {
            await page.screenshot({ path: 'e2e/screenshots/item_duplicate_menu.png', fullPage: true });
          }
        }
      }
    });

    test('should archive/delete item', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      const itemCard = page.locator('[data-type="question"], [data-type="dashboard"]').first();
      if (await itemCard.isVisible({ timeout: 3000 })) {
        await itemCard.hover();

        const menuBtn = itemCard.locator('button[aria-label="More"]');
        if (await menuBtn.isVisible({ timeout: 2000 })) {
          await menuBtn.click();

          const deleteOption = page.locator('[role="menuitem"]:has-text("Delete"), [role="menuitem"]:has-text("Archive")');
          if (await deleteOption.isVisible({ timeout: 2000 })) {
            await page.screenshot({ path: 'e2e/screenshots/item_delete_menu.png', fullPage: true });
          }
        }
      }
    });
  });

  test.describe('View Options', () => {
    test('should switch between grid and list view', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Look for view toggle
      const viewToggle = page.locator('[data-testid="view-toggle"], button[aria-label="Grid view"], button[aria-label="List view"]');
      if (await viewToggle.first().isVisible({ timeout: 3000 })) {
        await viewToggle.first().click();
        await page.waitForTimeout(300);

        await page.screenshot({ path: 'e2e/screenshots/browse_view_toggle.png', fullPage: true });
      }
    });
  });

  test.describe('UI Fidelity', () => {
    test('should have correct browse page layout', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Check header styling
      const header = page.locator('h2:has-text("Browse")');
      const fontWeight = await header.evaluate(el => getComputedStyle(el).fontWeight);
      expect(parseInt(fontWeight)).toBeGreaterThanOrEqual(600);

      // Check tabs styling
      const tabs = page.locator('[role="tablist"], .mantine-Tabs-list');
      if (await tabs.isVisible()) {
        const display = await tabs.evaluate(el => getComputedStyle(el).display);
        expect(display).toBe('flex');
      }

      await page.screenshot({ path: 'e2e/screenshots/browse_layout.png', fullPage: true });
    });

    test('should have correct card grid styling', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.waitForSelector('h2:has-text("Browse")');

      // Check card grid
      const cardGrid = page.locator('[data-testid="items-grid"], .mantine-SimpleGrid-root, .items-grid');
      if (await cardGrid.isVisible({ timeout: 3000 })) {
        const display = await cardGrid.evaluate(el => getComputedStyle(el).display);
        expect(display === 'grid' || display === 'flex').toBeTruthy();
      }

      await page.screenshot({ path: 'e2e/screenshots/browse_card_grid.png', fullPage: true });
    });
  });
});

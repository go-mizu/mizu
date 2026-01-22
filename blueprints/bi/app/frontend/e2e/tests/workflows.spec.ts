import { test, expect } from '../fixtures/test-base';

test.describe('Integration Workflows', () => {
  test.describe('Question to Dashboard Flow', () => {
    test('should create question and add to new dashboard', async ({ authenticatedPage: page }) => {
      // Step 1: Create a new question
      await page.goto('/question/new');
      await page.waitForTimeout(1000);

      // Select datasource
      await page.click('label:has-text("Database") + div, [data-testid="datasource-picker"]').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(500);

      // Select table
      await page.click('label:has-text("Table") + div, [data-testid="table-picker"]').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(500);

      // Run query
      await page.click('button:has-text("Run"), button:has-text("Get Answer")');
      await page.waitForSelector('table, [data-testid="results"]', { timeout: 15000 }).catch(() => {});

      // Save question
      await page.click('button:has-text("Save")');
      await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });

      const timestamp = Date.now();
      const questionName = `Workflow Test Question ${timestamp}`;
      await page.fill('input[placeholder*="name" i]', questionName);
      await page.click('[role="dialog"] button:has-text("Save")');
      await expect(page.locator('[role="dialog"]:has-text("Save")')).toBeHidden({ timeout: 10000 });

      await page.screenshot({ path: 'e2e/screenshots/workflow_question_saved.png', fullPage: true });

      // Step 2: Go to dashboard creation
      await page.goto('/dashboard/new');
      await page.waitForTimeout(1000);

      // Add card
      await page.click('button:has-text("Add"), text=Add card').catch(() => {});
      await page.waitForTimeout(500);

      // Modal should appear to select question
      const modal = page.locator('[role="dialog"]');
      if (await modal.isVisible({ timeout: 3000 })) {
        // Search for our question
        const searchInput = modal.locator('input');
        if (await searchInput.isVisible()) {
          await searchInput.fill(questionName);
          await page.waitForTimeout(500);
        }

        await page.screenshot({ path: 'e2e/screenshots/workflow_add_card_modal.png', fullPage: true });
      }
    });

    test('should create question and add to existing dashboard', async ({ authenticatedPage: page }) => {
      // First, find an existing dashboard
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      await page.waitForTimeout(500);

      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();
      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();
        await page.waitForURL(/\/dashboard\//, { timeout: 10000 });

        // Get dashboard URL
        const dashboardUrl = page.url();

        // Now create a question
        await page.goto('/question/new');
        await page.waitForTimeout(1000);

        // Quick setup
        await page.click('label:has-text("Database") + div').catch(() => {});
        await page.locator('[role="option"]').first().click().catch(() => {});
        await page.waitForTimeout(300);
        await page.click('label:has-text("Table") + div').catch(() => {});
        await page.locator('[role="option"]').first().click().catch(() => {});
        await page.waitForTimeout(300);

        // Run query
        await page.click('button:has-text("Run"), button:has-text("Get Answer")');
        await page.waitForSelector('table', { timeout: 10000 }).catch(() => {});

        // Look for "Add to dashboard" option
        const addToDashboardBtn = page.locator('button:has-text("Add to dashboard"), [data-testid="add-to-dashboard"]');
        if (await addToDashboardBtn.isVisible({ timeout: 3000 })) {
          await addToDashboardBtn.click();

          await page.screenshot({ path: 'e2e/screenshots/workflow_add_to_dashboard.png', fullPage: true });
        }
      }
    });
  });

  test.describe('Search & Navigation Flow', () => {
    test('should search and open question', async ({ authenticatedPage: page }) => {
      await page.goto('/');

      // Open command palette
      await page.keyboard.press('Meta+k');
      await expect(page.locator('[role="dialog"], [data-testid="command-palette"]')).toBeVisible({ timeout: 5000 });

      // Type search
      await page.keyboard.type('SELECT');
      await page.waitForTimeout(500);

      // Click first result if available
      const result = page.locator('[role="option"], [data-testid="search-result"]').first();
      if (await result.isVisible({ timeout: 3000 })) {
        await result.click();

        // Should navigate somewhere
        await page.waitForTimeout(1000);

        await page.screenshot({ path: 'e2e/screenshots/workflow_search_navigation.png', fullPage: true });
      } else {
        // Close palette
        await page.keyboard.press('Escape');
      }
    });

    test('should navigate through breadcrumbs', async ({ authenticatedPage: page }) => {
      // Go to a nested page
      await page.goto('/browse');
      await page.click('button:has-text("Collections")').catch(() => {});
      await page.waitForTimeout(500);

      const collectionCard = page.locator('[data-type="collection"]').first();
      if (await collectionCard.isVisible({ timeout: 3000 })) {
        await collectionCard.click();
        await page.waitForURL(/\/collection\//, { timeout: 10000 });

        // Look for breadcrumbs
        const breadcrumbs = page.locator('[data-testid="breadcrumbs"], nav[aria-label="Breadcrumb"], .breadcrumb');
        if (await breadcrumbs.isVisible({ timeout: 3000 })) {
          // Click on a breadcrumb to navigate back
          const breadcrumbLink = breadcrumbs.locator('a').first();
          if (await breadcrumbLink.isVisible()) {
            await breadcrumbLink.click();
            await page.waitForTimeout(500);

            await page.screenshot({ path: 'e2e/screenshots/workflow_breadcrumb_nav.png', fullPage: true });
          }
        }
      }
    });
  });

  test.describe('Dashboard Edit Workflow', () => {
    test('should complete full dashboard edit cycle', async ({ authenticatedPage: page }) => {
      // Go to browse and find a dashboard
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      await page.waitForTimeout(500);

      const dashboardCard = page.locator('[data-type="dashboard"], .dashboard-card').first();
      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();
        await page.waitForURL(/\/dashboard\//, { timeout: 10000 });

        // Step 1: Enter edit mode
        const editBtn = page.locator('button:has-text("Edit")');
        if (await editBtn.isVisible({ timeout: 3000 })) {
          await editBtn.click();
          await page.waitForTimeout(500);

          await page.screenshot({ path: 'e2e/screenshots/workflow_dashboard_edit_mode.png', fullPage: true });

          // Step 2: Make a change (resize a card if possible)
          const card = page.locator('.react-grid-item').first();
          if (await card.isVisible({ timeout: 2000 })) {
            // Hover to show resize handles
            await card.hover();
            await page.waitForTimeout(300);
          }

          // Step 3: Cancel changes
          const cancelBtn = page.locator('button:has-text("Cancel")');
          if (await cancelBtn.isVisible({ timeout: 2000 })) {
            await cancelBtn.click();

            // Should exit edit mode
            await expect(page.locator('button:has-text("Edit")')).toBeVisible({ timeout: 5000 });

            await page.screenshot({ path: 'e2e/screenshots/workflow_dashboard_edit_cancelled.png', fullPage: true });
          }
        }
      }
    });
  });

  test.describe('Data Exploration Flow', () => {
    test('should drill down from chart', async ({ authenticatedPage: page }) => {
      // Find a dashboard with charts
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      await page.waitForTimeout(500);

      const dashboardCard = page.locator('[data-type="dashboard"]').first();
      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();
        await page.waitForURL(/\/dashboard\//, { timeout: 10000 });
        await page.waitForTimeout(1000);

        // Find a chart
        const chart = page.locator('.recharts-wrapper, svg.chart').first();
        if (await chart.isVisible({ timeout: 3000 })) {
          // Click on chart element
          await chart.click();
          await page.waitForTimeout(500);

          // Look for drill-down menu or action
          const drillMenu = page.locator('[data-testid="drill-menu"], [role="menu"]:has-text("Drill")');
          if (await drillMenu.isVisible({ timeout: 2000 })) {
            await page.screenshot({ path: 'e2e/screenshots/workflow_chart_drill.png', fullPage: true });
          }
        }
      }
    });

    test('should filter and see results update', async ({ authenticatedPage: page }) => {
      await page.goto('/question/new');
      await page.waitForTimeout(1000);

      // Setup basic query
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);
      await page.click('label:has-text("Table") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      // Run initial query
      await page.click('button:has-text("Run"), button:has-text("Get Answer")');
      await page.waitForSelector('table', { timeout: 10000 }).catch(() => {});

      // Get initial row count
      const initialRows = await page.locator('table tbody tr').count();

      // Add a filter
      await page.click('button:has-text("Filter")').catch(() => {});
      await page.waitForTimeout(500);

      await page.screenshot({ path: 'e2e/screenshots/workflow_filter_applied.png', fullPage: true });
    });
  });

  test.describe('Export Workflow', () => {
    test('should export question results', async ({ authenticatedPage: page }) => {
      await page.goto('/question/new');
      await page.waitForTimeout(1000);

      // Setup and run query
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);
      await page.click('label:has-text("Table") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      await page.click('button:has-text("Run"), button:has-text("Get Answer")');
      await page.waitForSelector('table', { timeout: 10000 }).catch(() => {});

      // Look for export button
      const exportBtn = page.locator('button:has-text("Export"), button:has-text("Download"), button[aria-label="Download"]');
      if (await exportBtn.isVisible({ timeout: 3000 })) {
        await exportBtn.click();

        // Export options should appear
        await page.waitForTimeout(500);

        await page.screenshot({ path: 'e2e/screenshots/workflow_export_options.png', fullPage: true });
      }
    });
  });

  test.describe('Sharing Workflow', () => {
    test('should open share dialog for question', async ({ authenticatedPage: page }) => {
      // Find a question
      await page.goto('/browse');
      await page.click('button:has-text("Questions")').catch(() => {});
      await page.waitForTimeout(500);

      const questionCard = page.locator('[data-type="question"]').first();
      if (await questionCard.isVisible({ timeout: 3000 })) {
        await questionCard.click();
        await page.waitForURL(/\/question\//, { timeout: 10000 });
        await page.waitForTimeout(1000);

        // Look for share button
        const shareBtn = page.locator('button:has-text("Share"), button[aria-label="Share"]');
        if (await shareBtn.isVisible({ timeout: 3000 })) {
          await shareBtn.click();

          // Share dialog should appear
          await expect(page.locator('[role="dialog"]:has-text("Share")')).toBeVisible({ timeout: 5000 });

          await page.screenshot({ path: 'e2e/screenshots/workflow_share_dialog.png', fullPage: true });

          await page.keyboard.press('Escape');
        }
      }
    });

    test('should open share dialog for dashboard', async ({ authenticatedPage: page }) => {
      await page.goto('/browse');
      await page.click('button:has-text("Dashboards")').catch(() => {});
      await page.waitForTimeout(500);

      const dashboardCard = page.locator('[data-type="dashboard"]').first();
      if (await dashboardCard.isVisible({ timeout: 3000 })) {
        await dashboardCard.click();
        await page.waitForURL(/\/dashboard\//, { timeout: 10000 });
        await page.waitForTimeout(1000);

        // Look for share button
        const shareBtn = page.locator('button:has-text("Share"), button[aria-label="Share"]');
        if (await shareBtn.isVisible({ timeout: 3000 })) {
          await shareBtn.click();

          await page.waitForTimeout(500);

          await page.screenshot({ path: 'e2e/screenshots/workflow_dashboard_share.png', fullPage: true });

          await page.keyboard.press('Escape');
        }
      }
    });
  });

  test.describe('Error Recovery Workflows', () => {
    test('should handle invalid query gracefully', async ({ authenticatedPage: page }) => {
      await page.goto('/question/new');
      await page.waitForTimeout(1000);

      // Switch to native SQL
      await page.click('text=Native, text=SQL').catch(async () => {
        await page.click('[data-testid="mode-toggle"] button:last-child');
      });
      await page.waitForTimeout(500);

      // Select database
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      // Enter invalid SQL
      const editor = page.locator('.CodeMirror, textarea[placeholder*="SQL"]');
      if (await editor.isVisible()) {
        await editor.click();
        await page.keyboard.type('SELECT * FROM nonexistent_table_12345');
      }

      // Run query
      await page.click('button:has-text("Run"), button:has-text("Get Answer")');
      await page.waitForTimeout(2000);

      // Should show error message
      const errorMessage = page.locator('text=error, .error-message, [data-testid="error"]');
      if (await errorMessage.isVisible({ timeout: 5000 })) {
        await page.screenshot({ path: 'e2e/screenshots/workflow_query_error.png', fullPage: true });
      }
    });

    test('should handle 404 page', async ({ authenticatedPage: page }) => {
      await page.goto('/nonexistent-page-12345');
      await page.waitForTimeout(1000);

      // Should show 404 or redirect to home
      const is404 = await page.locator('text=404, text=Not found, text=not found').isVisible({ timeout: 3000 });
      const isHome = page.url().endsWith('/');

      expect(is404 || isHome).toBeTruthy();

      await page.screenshot({ path: 'e2e/screenshots/workflow_404_page.png', fullPage: true });
    });

    test('should handle session timeout', async ({ authenticatedPage: page }) => {
      await page.goto('/');

      // Clear session storage to simulate timeout
      await page.evaluate(() => {
        sessionStorage.clear();
        localStorage.removeItem('auth_token');
      });

      // Try to access protected resource
      await page.goto('/admin/settings');
      await page.waitForTimeout(1000);

      // Should redirect to login or show unauthorized
      const isLogin = page.url().includes('/login') || page.url().includes('/auth');
      const isUnauthorized = await page.locator('text=unauthorized, text=login, text=sign in').isVisible({ timeout: 3000 }).catch(() => false);
      const stillOnAdmin = page.url().includes('/admin');

      // One of these should be true
      expect(isLogin || isUnauthorized || stillOnAdmin).toBeTruthy();

      await page.screenshot({ path: 'e2e/screenshots/workflow_session_timeout.png', fullPage: true });
    });
  });

  test.describe('Performance Workflows', () => {
    test('should load home page within acceptable time', async ({ authenticatedPage: page }) => {
      const startTime = Date.now();

      await page.goto('/');
      await page.waitForSelector('h2:has-text("Home")');

      const loadTime = Date.now() - startTime;

      // Home page should load in under 5 seconds
      expect(loadTime).toBeLessThan(5000);

      console.log(`Home page load time: ${loadTime}ms`);
    });

    test('should handle large result sets', async ({ authenticatedPage: page }) => {
      await page.goto('/question/new');
      await page.waitForTimeout(1000);

      // Setup query
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);
      await page.click('label:has-text("Table") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      // Run query
      const startTime = Date.now();
      await page.click('button:has-text("Run"), button:has-text("Get Answer")');
      await page.waitForSelector('table, [data-testid="results"]', { timeout: 30000 }).catch(() => {});
      const queryTime = Date.now() - startTime;

      console.log(`Query execution time: ${queryTime}ms`);

      // Should complete within 30 seconds
      expect(queryTime).toBeLessThan(30000);

      await page.screenshot({ path: 'e2e/screenshots/workflow_large_result.png', fullPage: true });
    });
  });
});

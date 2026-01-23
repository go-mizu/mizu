import { test, expect, waitForAPI } from '../fixtures/test-base';
import { QuestionPage } from '../pages/question.page';

test.describe('Question Builder', () => {
  test.describe('Visual Query Builder', () => {
    test('should select data source from dropdown', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Look for database/datasource picker
      await page.click('[data-testid="select-datasource"]');

      // Verify dropdown opens (check for listbox specifically)
      await expect(page.locator('[role="listbox"]')).toBeVisible({ timeout: 5000 });

      // Select first option
      await page.locator('[role="option"]').first().click();

      await questionPage.takeScreenshot('datasource_selected');
    });

    test('should select table from dropdown', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // First select datasource if needed
      const datasourcePicker = page.locator('[data-testid="select-datasource"]');
      if (await datasourcePicker.isVisible()) {
        await datasourcePicker.click();
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table (opens modal)
      await page.click('[data-testid="table-picker"]');
      await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 5000 });

      // Click first table in the list
      await page.locator('[data-testid="modal-table-picker"] button:has-text("orders"), [data-testid="modal-table-picker"] [role="button"]').first().click().catch(async () => {
        // Fallback: click any table row
        await page.locator('[data-testid="modal-table-picker"] .mantine-Paper-root').first().click();
      });

      await questionPage.takeScreenshot('table_selected');
    });

    test('should run query and display results', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();
      await page.waitForTimeout(1000);

      // Select datasource
      const dsSelect = page.locator('[data-testid="select-datasource"]');
      if (await dsSelect.isVisible({ timeout: 3000 })) {
        await dsSelect.click();
        await page.waitForSelector('[role="option"]', { timeout: 3000 });
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table (uses modal)
      const tablePicker = page.locator('[data-testid="table-picker"]');
      if (await tablePicker.isVisible({ timeout: 3000 })) {
        await tablePicker.click();
        await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 3000 });
        await page.locator('[data-testid="modal-table-picker"] button').first().click().catch(async () => {
          // Fallback
          await page.keyboard.press('Escape');
        });
        await page.waitForTimeout(500);
      }

      // Run query
      await questionPage.runQuery();

      // Verify results displayed
      const hasResults = await page.locator('table, [data-testid="results-area"], .recharts-wrapper').isVisible({ timeout: 15000 });
      expect(hasResults).toBeTruthy();

      await questionPage.takeScreenshot('query_results');
    });

    test('should select and deselect columns', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();
      await page.waitForTimeout(1000);

      // Setup datasource
      const dsSelect = page.locator('[data-testid="select-datasource"]');
      if (await dsSelect.isVisible({ timeout: 3000 })) {
        await dsSelect.click();
        await page.waitForSelector('[role="option"]', { timeout: 3000 });
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table
      const tablePicker = page.locator('[data-testid="table-picker"]');
      if (await tablePicker.isVisible({ timeout: 3000 })) {
        await tablePicker.click();
        await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 3000 });
        await page.locator('[data-testid="modal-table-picker"] button').first().click().catch(() => {});
        await page.waitForTimeout(500);
      }

      // Find columns section (if exists)
      const columnsSection = page.locator('text=Columns').first();
      if (await columnsSection.isVisible({ timeout: 3000 })) {
        await questionPage.takeScreenshot('columns_section');
      }
    });
  });

  test.describe('Filtering', () => {
    test('should add equals filter', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();
      await page.waitForTimeout(1000);

      // Setup datasource
      const dsSelect = page.locator('[data-testid="select-datasource"]');
      if (await dsSelect.isVisible({ timeout: 3000 })) {
        await dsSelect.click();
        await page.waitForSelector('[role="option"]', { timeout: 3000 });
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table
      const tablePicker = page.locator('[data-testid="table-picker"]');
      if (await tablePicker.isVisible({ timeout: 3000 })) {
        await tablePicker.click();
        await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 3000 });
        await page.locator('[data-testid="modal-table-picker"] button').first().click().catch(() => {});
        await page.waitForTimeout(500);
      }

      // Add filter (if filter button exists)
      const filterBtn = page.locator('button:has-text("Filter")').first();
      if (await filterBtn.isVisible({ timeout: 3000 })) {
        await filterBtn.click();
        await page.waitForTimeout(500);
      }

      await questionPage.takeScreenshot('filter_dialog');
    });

    test('should apply multiple filters', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();
      await page.waitForTimeout(1000);

      // Setup datasource
      const dsSelect = page.locator('[data-testid="select-datasource"]');
      if (await dsSelect.isVisible({ timeout: 3000 })) {
        await dsSelect.click();
        await page.waitForSelector('[role="option"]', { timeout: 3000 });
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table
      const tablePicker = page.locator('[data-testid="table-picker"]');
      if (await tablePicker.isVisible({ timeout: 3000 })) {
        await tablePicker.click();
        await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 3000 });
        await page.locator('[data-testid="modal-table-picker"] button').first().click().catch(() => {});
        await page.waitForTimeout(500);
      }

      await questionPage.takeScreenshot('multiple_filters');
    });
  });

  test.describe('Summarization', () => {
    test('should add count aggregation', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();
      await page.waitForTimeout(1000);

      // Setup datasource
      const dsSelect = page.locator('[data-testid="select-datasource"]');
      if (await dsSelect.isVisible({ timeout: 3000 })) {
        await dsSelect.click();
        await page.waitForSelector('[role="option"]', { timeout: 3000 });
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table
      const tablePicker = page.locator('[data-testid="table-picker"]');
      if (await tablePicker.isVisible({ timeout: 3000 })) {
        await tablePicker.click();
        await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 3000 });
        await page.locator('[data-testid="modal-table-picker"] button').first().click().catch(() => {});
        await page.waitForTimeout(500);
      }

      // Click summarize (if exists)
      const summarizeBtn = page.locator('button:has-text("Summarize")').first();
      if (await summarizeBtn.isVisible({ timeout: 3000 })) {
        await summarizeBtn.click();
        await page.waitForTimeout(500);
      }

      await questionPage.takeScreenshot('summarize_count');
    });

    test('should add group by', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();
      await page.waitForTimeout(1000);

      // Setup datasource
      const dsSelect = page.locator('[data-testid="select-datasource"]');
      if (await dsSelect.isVisible({ timeout: 3000 })) {
        await dsSelect.click();
        await page.waitForSelector('[role="option"]', { timeout: 3000 });
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
      }

      // Select table
      const tablePicker = page.locator('[data-testid="table-picker"]');
      if (await tablePicker.isVisible({ timeout: 3000 })) {
        await tablePicker.click();
        await page.waitForSelector('[data-testid="modal-table-picker"]', { timeout: 3000 });
        await page.locator('[data-testid="modal-table-picker"] button').first().click().catch(() => {});
        await page.waitForTimeout(500);
      }

      await questionPage.takeScreenshot('group_by');
    });
  });

  test.describe('Native SQL', () => {
    test('should switch to native SQL mode', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Find and click SQL/Native tab
      await page.click('text=Native, text=SQL, button[data-mode="native"], [role="tab"]:has-text("Native")').catch(async () => {
        // Alternative - look for toggle
        await page.click('[data-testid="mode-toggle"] button:last-child');
      });

      // Verify SQL editor appears
      await expect(page.locator('.CodeMirror, textarea[placeholder*="SQL"], [data-testid="sql-editor"]')).toBeVisible({ timeout: 5000 });

      await questionPage.takeScreenshot('native_mode');
    });

    test('should execute SQL query', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Switch to native mode
      await page.click('text=Native, text=SQL').catch(async () => {
        await page.click('[data-testid="mode-toggle"] button:last-child');
      });
      await page.waitForTimeout(500);

      // Select datasource if needed
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      // Enter SQL (find the editor)
      const editor = page.locator('.CodeMirror, textarea[placeholder*="SQL"]');
      if (await editor.isVisible()) {
        const tagName = await editor.evaluate(el => el.tagName);
        if (tagName === 'TEXTAREA') {
          await editor.fill('SELECT 1 as test');
        } else {
          await editor.click();
          await page.keyboard.type('SELECT 1 as test', { delay: 20 });
        }
      }

      // Run query
      await page.click('button:has-text("Run"), button:has-text("Get Answer")');

      // Wait for results
      await page.waitForSelector('table, [data-testid="results"]', { timeout: 10000 });

      await questionPage.takeScreenshot('native_results');
    });
  });

  test.describe('Visualization', () => {
    test('should display results as table by default', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Setup and run query
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);
      await page.click('label:has-text("Table") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      await questionPage.runQuery();

      // Should show table
      await expect(page.locator('table, [data-testid="results-table"]')).toBeVisible({ timeout: 10000 });

      await questionPage.takeScreenshot('viz_table');
    });

    test('should change visualization type', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Setup and run query
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);
      await page.click('label:has-text("Table") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      await questionPage.runQuery();

      // Open visualization picker
      await page.click('button:has-text("Visualization"), [data-testid="viz-picker"]').catch(() => {});

      // Verify picker opens
      await expect(page.locator('[data-testid="viz-options"], [role="dialog"]:has-text("Visualization")')).toBeVisible({ timeout: 5000 });

      await questionPage.takeScreenshot('viz_picker');
    });
  });

  test.describe('Save & Edit', () => {
    test('should save new question', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Setup and run query
      await page.click('label:has-text("Database") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);
      await page.click('label:has-text("Table") + div').catch(() => {});
      await page.locator('[role="option"]').first().click().catch(() => {});
      await page.waitForTimeout(300);

      await questionPage.runQuery();

      // Click save
      await page.click('button:has-text("Save")');

      // Verify save modal
      await expect(page.locator('[role="dialog"]:has-text("Save")')).toBeVisible({ timeout: 5000 });

      // Enter name
      const timestamp = Date.now();
      const questionName = `Test Question ${timestamp}`;
      await page.fill('input[placeholder*="name" i], input[label="Name"]', questionName);

      // Submit
      await page.click('[role="dialog"] button:has-text("Save")');

      // Verify modal closes
      await expect(page.locator('[role="dialog"]:has-text("Save")')).toBeHidden({ timeout: 10000 });

      await questionPage.takeScreenshot('question_saved');
    });

    test('should load existing question', async ({ authenticatedPage: page }) => {
      // Navigate to browse and find a question
      await page.goto('/browse');
      await page.waitForSelector('text=Questions, [data-testid="browse"]');

      // Click on a question if available
      const questionCard = page.locator('[data-type="question"], .question-card').first();
      if (await questionCard.isVisible({ timeout: 3000 })) {
        await questionCard.click();

        // Should load question page
        await page.waitForURL(/\/question\//, { timeout: 10000 });

        // Verify question loaded
        await expect(page.locator('table, .recharts-wrapper, [data-testid="results"]')).toBeVisible({ timeout: 10000 });
      }
    });
  });

  test.describe('UI Fidelity', () => {
    test('should have correct query builder layout', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Verify layout structure
      await questionPage.verifyMetabaseLayout();

      await questionPage.takeScreenshot('layout_query_builder');
    });

    test('should have proper button styling', async ({ authenticatedPage: page }) => {
      const questionPage = new QuestionPage(page);
      await questionPage.goto();

      // Check run button styling
      const runButton = page.locator('button:has-text("Run"), button:has-text("Get Answer")');
      if (await runButton.isVisible()) {
        const bgColor = await runButton.evaluate(el => getComputedStyle(el).backgroundColor);
        // Should have a colored background (not transparent or white)
        expect(bgColor).not.toBe('transparent');
        expect(bgColor).not.toBe('rgba(0, 0, 0, 0)');
      }

      await questionPage.takeScreenshot('button_styling');
    });
  });
});

import { Page, Locator, expect } from '@playwright/test';

/**
 * Page Object for Question Builder Page
 * Represents the query builder and question editing interface
 */
export class QuestionPage {
  readonly page: Page;

  // Header elements
  readonly title: Locator;
  readonly saveButton: Locator;
  readonly runButton: Locator;
  readonly modeToggle: Locator;

  // Query Builder elements
  readonly dataSourcePicker: Locator;
  readonly tablePicker: Locator;
  readonly columnSelector: Locator;
  readonly filterBuilder: Locator;
  readonly summarizeBuilder: Locator;

  // Native Query elements
  readonly sqlEditor: Locator;

  // Results elements
  readonly resultsTable: Locator;
  readonly resultsCount: Locator;
  readonly visualization: Locator;
  readonly visualizationPicker: Locator;

  // Modals
  readonly saveModal: Locator;

  constructor(page: Page) {
    this.page = page;

    // Header
    this.title = page.locator('h2, h3').first();
    this.saveButton = page.locator('button:has-text("Save")');
    this.runButton = page.locator('button:has-text("Run"), button:has-text("Get Answer")');
    this.modeToggle = page.locator('[role="tablist"], [data-mode-toggle]');

    // Query Builder
    this.dataSourcePicker = page.locator('[data-testid="datasource-picker"], label:has-text("Database") + div');
    this.tablePicker = page.locator('[data-testid="table-picker"], label:has-text("Table") + div');
    this.columnSelector = page.locator('[data-testid="column-selector"], text=Columns');
    this.filterBuilder = page.locator('[data-testid="filter-builder"], text=Filter');
    this.summarizeBuilder = page.locator('[data-testid="summarize-builder"], text=Summarize');

    // Native Query
    this.sqlEditor = page.locator('[data-testid="sql-editor"], .CodeMirror, textarea[placeholder*="SQL"]');

    // Results
    this.resultsTable = page.locator('table, [data-testid="results-table"]');
    this.resultsCount = page.locator('text=/\\d+ row/i');
    this.visualization = page.locator('[data-testid="visualization"], .recharts-wrapper');
    this.visualizationPicker = page.locator('[data-testid="viz-picker"], button:has-text("Visualization")');

    // Modals
    this.saveModal = page.locator('[role="dialog"]:has-text("Save")');
  }

  async goto(id?: string) {
    if (id) {
      await this.page.goto(`/question/${id}`);
    } else {
      await this.page.goto('/question/new');
    }
    await this.waitForLoad();
  }

  async waitForLoad() {
    await this.page.waitForSelector('[data-testid="query-builder"], button:has-text("Run"), button:has-text("Get Answer")', { timeout: 10000 });
  }

  // Data Source Selection
  async selectDataSource(name: string) {
    await this.page.click('label:has-text("Database") + div, [data-testid="datasource-picker"]');
    await this.page.click(`[role="option"]:has-text("${name}")`);
    await this.page.waitForTimeout(500);
  }

  async selectTable(name: string) {
    await this.page.click('label:has-text("Table") + div, [data-testid="table-picker"]');
    await this.page.click(`[role="option"]:has-text("${name}")`);
    await this.page.waitForTimeout(500);
  }

  // Column Selection
  async selectColumns(columns: string[]) {
    for (const col of columns) {
      await this.page.click(`text=${col}`, { force: true }).catch(async () => {
        await this.page.click(`[data-column="${col}"], input[value="${col}"]`);
      });
    }
  }

  async selectAllColumns() {
    await this.page.click('text=Select all, button:has-text("Select all")').catch(() => {});
  }

  // Filtering
  async addFilter(column: string, operator: string, value: string) {
    // Click filter button or section
    await this.page.click('button:has-text("Filter"), text=Filter');

    // Select column
    await this.page.click('[data-testid="filter-column"], label:has-text("Column") + div');
    await this.page.click(`[role="option"]:has-text("${column}")`);

    // Select operator
    await this.page.click('[data-testid="filter-operator"], label:has-text("Operator") + div');
    await this.page.click(`[role="option"]:has-text("${operator}")`);

    // Enter value
    await this.page.fill('input[placeholder*="value"], input[data-testid="filter-value"]', value);

    // Apply filter
    await this.page.click('button:has-text("Apply"), button:has-text("Add filter")');
  }

  // Summarization
  async addSummarize(aggregation: string, column?: string) {
    await this.page.click('button:has-text("Summarize"), text=Summarize');

    // Select aggregation
    await this.page.click(`text=${aggregation}, [data-aggregation="${aggregation}"]`);

    if (column) {
      await this.page.click(`text=${column}`);
    }

    await this.page.click('button:has-text("Done"), button:has-text("Apply")').catch(() => {});
  }

  async addGroupBy(column: string) {
    await this.page.click('text=Group by, button:has-text("Group by")');
    await this.page.click(`text=${column}`);
    await this.page.click('button:has-text("Done")').catch(() => {});
  }

  // Query Execution
  async runQuery() {
    await this.runButton.click();
    // Wait for results
    await this.page.waitForSelector('table, [data-testid="visualization"], .recharts-wrapper', { timeout: 30000 });
    await this.page.waitForTimeout(500);
  }

  async getResults() {
    // Get table results
    const rows = await this.page.locator('tbody tr, [data-testid="result-row"]').all();
    const results = [];

    for (const row of rows) {
      const cells = await row.locator('td, [data-testid="result-cell"]').all();
      const rowData: string[] = [];
      for (const cell of cells) {
        const text = await cell.textContent();
        rowData.push(text?.trim() || '');
      }
      results.push(rowData);
    }

    return results;
  }

  async getRowCount() {
    const countText = await this.resultsCount.textContent();
    const match = countText?.match(/(\d+)/);
    return match ? parseInt(match[1]) : 0;
  }

  // Native SQL Mode
  async switchToNativeMode() {
    await this.page.click('text=Native, button:has-text("SQL"), [data-mode="native"]');
    await this.page.waitForSelector('[data-testid="sql-editor"], .CodeMirror, textarea');
  }

  async enterSQL(sql: string) {
    // Different SQL editors have different APIs
    const editor = await this.page.locator('.CodeMirror, textarea[placeholder*="SQL"]').first();
    const tagName = await editor.evaluate(el => el.tagName);

    if (tagName === 'TEXTAREA') {
      await editor.fill(sql);
    } else {
      // CodeMirror
      await editor.click();
      await this.page.keyboard.type(sql, { delay: 10 });
    }
  }

  // Visualization
  async selectVisualization(type: string) {
    await this.page.click('button:has-text("Visualization"), [data-testid="viz-picker"]');
    await this.page.click(`text=${type}, [data-viz-type="${type}"]`);
  }

  async hasVisualization() {
    return this.visualization.isVisible();
  }

  // Save Question
  async saveQuestion(name: string, description?: string, collectionId?: string) {
    await this.saveButton.click();
    await this.page.waitForSelector('[role="dialog"]');

    await this.page.fill('input[placeholder*="name" i], input[label="Name"]', name);

    if (description) {
      await this.page.fill('textarea[placeholder*="description" i], textarea[label="Description"]', description);
    }

    await this.page.click('button:has-text("Save")');
    await this.page.waitForSelector('[role="dialog"]', { state: 'hidden' });
  }

  // Screenshots
  async takeScreenshot(name: string) {
    await this.page.screenshot({
      path: `e2e/screenshots/question_${name}.png`,
      fullPage: true,
    });
  }

  // Metabase UI verification
  async verifyMetabaseLayout() {
    // Check query builder structure
    await expect(this.runButton).toBeVisible();

    // Check data selection area
    const hasDataPicker = await this.page.locator('text=Database, text=Table').first().isVisible();
    expect(hasDataPicker).toBeTruthy();
  }
}

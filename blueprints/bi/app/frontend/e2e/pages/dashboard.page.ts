import { Page, Locator, expect } from '@playwright/test';

/**
 * Page Object for Dashboard Page
 * Represents the dashboard viewing and editing interface
 */
export class DashboardPage {
  readonly page: Page;

  // Header elements
  readonly title: Locator;
  readonly editButton: Locator;
  readonly saveButton: Locator;
  readonly addCardButton: Locator;
  readonly refreshButton: Locator;
  readonly fullscreenButton: Locator;

  // Dashboard content
  readonly dashboardGrid: Locator;
  readonly dashboardCards: Locator;
  readonly emptyState: Locator;

  // Filters
  readonly filterBar: Locator;
  readonly filterPills: Locator;

  // Tabs
  readonly tabList: Locator;
  readonly addTabButton: Locator;

  // Modals
  readonly addCardModal: Locator;
  readonly saveModal: Locator;
  readonly settingsModal: Locator;

  constructor(page: Page) {
    this.page = page;

    // Header
    this.title = page.locator('h2, h1, [data-testid="dashboard-title"]');
    this.editButton = page.locator('button:has-text("Edit"), button:has-text("Pencil")');
    this.saveButton = page.locator('button:has-text("Save")');
    this.addCardButton = page.locator('button:has-text("Add"), button:has-text("Add card")');
    this.refreshButton = page.locator('button[aria-label="Refresh"], button:has-text("Refresh")');
    this.fullscreenButton = page.locator('button[aria-label="Fullscreen"], button:has-text("Fullscreen")');

    // Dashboard content
    this.dashboardGrid = page.locator('.react-grid-layout, [data-testid="dashboard-grid"]');
    this.dashboardCards = page.locator('.react-grid-item, [data-testid="dashboard-card"]');
    this.emptyState = page.locator('text=Add a card, text=This dashboard is empty');

    // Filters
    this.filterBar = page.locator('[data-testid="filter-bar"], .dashboard-filters');
    this.filterPills = page.locator('[data-testid="filter-pill"], .filter-pill');

    // Tabs
    this.tabList = page.locator('[role="tablist"], .dashboard-tabs');
    this.addTabButton = page.locator('button:has-text("Add tab")');

    // Modals
    this.addCardModal = page.locator('[role="dialog"]:has-text("Add card")');
    this.saveModal = page.locator('[role="dialog"]:has-text("Save")');
    this.settingsModal = page.locator('[role="dialog"]:has-text("Settings")');
  }

  async goto(id?: string) {
    if (id) {
      await this.page.goto(`/dashboard/${id}`);
    } else {
      await this.page.goto('/dashboard/new');
    }
    await this.waitForLoad();
  }

  async waitForLoad() {
    await this.page.waitForSelector('[data-testid="dashboard-grid"], .react-grid-layout, h2, h1', { timeout: 10000 });
    // Wait for cards to load
    await this.page.waitForTimeout(500);
  }

  // Dashboard Information
  async getTitle() {
    return this.title.textContent();
  }

  async getCardCount() {
    return this.dashboardCards.count();
  }

  async getCardTitles() {
    const cards = await this.dashboardCards.all();
    const titles: string[] = [];
    for (const card of cards) {
      const title = await card.locator('h3, h4, [data-testid="card-title"]').textContent();
      if (title) titles.push(title.trim());
    }
    return titles;
  }

  // Edit Mode
  async enterEditMode() {
    await this.editButton.click();
    await this.page.waitForSelector('button:has-text("Save"), button:has-text("Done")');
  }

  async exitEditMode() {
    await this.saveButton.click();
    await this.page.waitForSelector('button:has-text("Edit")');
  }

  async cancelEdit() {
    await this.page.click('button:has-text("Cancel")');
  }

  // Add Cards
  async openAddCardModal() {
    await this.addCardButton.click();
    await this.page.waitForSelector('[role="dialog"]');
  }

  async addQuestionCard(questionName: string) {
    await this.openAddCardModal();
    await this.page.click(`text=${questionName}`);
    await this.page.click('button:has-text("Add"), button:has-text("Select")');
    await this.page.waitForSelector('[role="dialog"]', { state: 'hidden' });
  }

  async addNewQuestionCard() {
    await this.openAddCardModal();
    await this.page.click('text=New question, button:has-text("Create new")');
  }

  // Card Management
  async clickCard(index: number) {
    const cards = await this.dashboardCards.all();
    if (cards[index]) {
      await cards[index].click();
    }
  }

  async resizeCard(index: number, width: number, height: number) {
    const cards = await this.dashboardCards.all();
    if (cards[index]) {
      const resizeHandle = cards[index].locator('.react-resizable-handle');
      await resizeHandle.dragTo(resizeHandle, { targetPosition: { x: width, y: height } });
    }
  }

  async moveCard(index: number, x: number, y: number) {
    const cards = await this.dashboardCards.all();
    if (cards[index]) {
      await cards[index].dragTo(this.dashboardGrid, { targetPosition: { x, y } });
    }
  }

  async removeCard(index: number) {
    const cards = await this.dashboardCards.all();
    if (cards[index]) {
      await cards[index].hover();
      await cards[index].locator('button[aria-label="Remove"], button:has-text("Remove")').click();
    }
  }

  // Filters
  async addFilter(column: string, values: string[]) {
    await this.page.click('button:has-text("Add filter")');
    await this.page.click(`text=${column}`);
    for (const value of values) {
      await this.page.click(`text=${value}`);
    }
    await this.page.click('button:has-text("Apply"), button:has-text("Update filter")');
  }

  async clearFilter(filterName: string) {
    await this.page.click(`[data-filter="${filterName}"] button[aria-label="Clear"]`);
  }

  async clearAllFilters() {
    await this.page.click('button:has-text("Clear all")');
  }

  // Tabs
  async getTabCount() {
    return this.page.locator('[role="tab"]').count();
  }

  async addTab(name: string) {
    await this.addTabButton.click();
    await this.page.fill('input[placeholder*="tab name" i]', name);
    await this.page.click('button:has-text("Add"), button:has-text("Create")');
  }

  async switchTab(name: string) {
    await this.page.click(`[role="tab"]:has-text("${name}")`);
    await this.waitForLoad();
  }

  // Auto-refresh
  async setAutoRefresh(interval: string) {
    await this.page.click('button[aria-label="Auto-refresh"], button:has-text("Auto-refresh")');
    await this.page.click(`text=${interval}`);
  }

  async disableAutoRefresh() {
    await this.setAutoRefresh('Off');
  }

  // Fullscreen
  async enterFullscreen() {
    await this.fullscreenButton.click();
    await this.page.waitForSelector('.fullscreen, [data-fullscreen="true"]');
  }

  async exitFullscreen() {
    await this.page.keyboard.press('Escape');
  }

  // Save
  async saveDashboard(name?: string, description?: string) {
    await this.saveButton.click();

    if (name) {
      await this.page.waitForSelector('[role="dialog"]');
      await this.page.fill('input[placeholder*="name" i]', name);
      if (description) {
        await this.page.fill('textarea[placeholder*="description" i]', description);
      }
      await this.page.click('button:has-text("Save")');
    }

    await this.page.waitForSelector('[role="dialog"]', { state: 'hidden', timeout: 5000 }).catch(() => {});
  }

  // Screenshots
  async takeScreenshot(name: string) {
    await this.page.screenshot({
      path: `e2e/screenshots/dashboard_${name}.png`,
      fullPage: true,
    });
  }

  // Metabase UI verification
  async verifyMetabaseLayout() {
    // Check header structure
    await expect(this.title).toBeVisible();

    // Check grid layout
    const hasGrid = await this.dashboardGrid.isVisible().catch(() => false);
    const hasEmpty = await this.emptyState.isVisible().catch(() => false);
    expect(hasGrid || hasEmpty).toBeTruthy();
  }

  // Card visualization verification
  async verifyCardRendered(index: number) {
    const cards = await this.dashboardCards.all();
    expect(cards.length).toBeGreaterThan(index);

    const card = cards[index];
    // Check for either a chart or table
    const hasVisualization = await card.locator('.recharts-wrapper, table, [data-testid="visualization"]').isVisible().catch(() => false);
    return hasVisualization;
  }
}

import { Page, Locator, expect } from '@playwright/test';

/**
 * Page Object for Home Page
 * Represents the main landing page after login
 */
export class HomePage {
  readonly page: Page;

  // Header elements
  readonly title: Locator;
  readonly searchButton: Locator;
  readonly newButton: Locator;

  // Menu items
  readonly newQuestionMenuItem: Locator;
  readonly newDashboardMenuItem: Locator;
  readonly newCollectionMenuItem: Locator;

  // Sections
  readonly pinnedSection: Locator;
  readonly recentSection: Locator;
  readonly analyticsSection: Locator;
  readonly startHereSection: Locator;

  // Stat cards
  readonly questionsCount: Locator;
  readonly dashboardsCount: Locator;
  readonly collectionsCount: Locator;
  readonly databasesCount: Locator;

  constructor(page: Page) {
    this.page = page;

    // Header
    this.title = page.locator('h2:has-text("Home")');
    this.searchButton = page.locator('button:has-text("Search")');
    this.newButton = page.getByRole('button', { name: 'New', exact: true });

    // Menu items
    this.newQuestionMenuItem = page.locator('[role="menuitem"]:has-text("Question")');
    this.newDashboardMenuItem = page.locator('[role="menuitem"]:has-text("Dashboard")');
    this.newCollectionMenuItem = page.locator('[role="menuitem"]:has-text("Collection")');

    // Sections
    this.pinnedSection = page.locator('text=Pinned').locator('..');
    this.recentSection = page.locator('text=Pick up where you left off').locator('..');
    this.analyticsSection = page.locator('text=Our analytics').locator('..');
    this.startHereSection = page.locator('text=Start here').locator('..');

    // Stat cards
    this.questionsCount = page.locator('text=Questions').locator('xpath=..').locator('[data-value], span:first-child');
    this.dashboardsCount = page.locator('text=Dashboards').locator('xpath=..').locator('[data-value], span:first-child');
    this.collectionsCount = page.locator('text=Collections').locator('xpath=..').locator('[data-value], span:first-child');
    this.databasesCount = page.locator('text=Databases').locator('xpath=..').locator('[data-value], span:first-child');
  }

  async goto() {
    await this.page.goto('/');
    await this.waitForLoad();
  }

  async waitForLoad() {
    await this.page.waitForSelector('h2:has-text("Home")', { timeout: 10000 });
    // Wait for data to load
    await this.page.waitForTimeout(500);
  }

  async openNewMenu() {
    await this.newButton.click();
    await this.page.waitForSelector('[role="menu"]');
  }

  async createNewQuestion() {
    await this.openNewMenu();
    await this.newQuestionMenuItem.click();
    await this.page.waitForURL('**/question/new');
  }

  async createNewDashboard() {
    await this.openNewMenu();
    await this.newDashboardMenuItem.click();
    await this.page.waitForURL('**/dashboard/new');
  }

  async createNewCollection() {
    await this.openNewMenu();
    await this.newCollectionMenuItem.click();
  }

  async openSearch() {
    await this.searchButton.click();
    await this.page.waitForSelector('[role="dialog"]');
  }

  async clickPinnedItem(name: string) {
    await this.page.click(`text=${name}`);
  }

  async clickRecentItem(name: string) {
    await this.page.click(`text=${name}`);
  }

  async getStats() {
    const stats = {
      questions: 0,
      dashboards: 0,
      collections: 0,
      databases: 0,
    };

    // Extract numbers from stat cards
    const cards = await this.page.locator('[data-testid="stat-card"], div:has(> span + span:has-text(/^\\d+$/))').all();

    for (const card of cards) {
      const text = await card.textContent();
      if (text?.includes('Questions')) {
        const match = text.match(/(\d+)/);
        if (match) stats.questions = parseInt(match[1]);
      } else if (text?.includes('Dashboards')) {
        const match = text.match(/(\d+)/);
        if (match) stats.dashboards = parseInt(match[1]);
      } else if (text?.includes('Collections')) {
        const match = text.match(/(\d+)/);
        if (match) stats.collections = parseInt(match[1]);
      } else if (text?.includes('Databases')) {
        const match = text.match(/(\d+)/);
        if (match) stats.databases = parseInt(match[1]);
      }
    }

    return stats;
  }

  async hasEmptyState() {
    return this.page.locator('text=Ready to explore your data').isVisible();
  }

  async hasPinnedItems() {
    return this.pinnedSection.isVisible();
  }

  async hasRecentItems() {
    return this.recentSection.isVisible();
  }

  // Metabase UI comparison helpers
  async verifyMetabaseLayout() {
    // Check header structure
    await expect(this.title).toBeVisible();
    await expect(this.searchButton).toBeVisible();
    await expect(this.newButton).toBeVisible();

    // Check section titles use uppercase letters and proper styling
    const sectionTitles = await this.page.locator('text=/^(PINNED|PICK UP|OUR ANALYTICS|START HERE)/i').all();
    expect(sectionTitles.length).toBeGreaterThanOrEqual(1);
  }

  async takeScreenshot(name: string) {
    await this.page.screenshot({
      path: `e2e/screenshots/home_${name}.png`,
      fullPage: true,
    });
  }
}

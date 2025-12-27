import { type Page, type Locator, expect } from '@playwright/test';

export class HomePage {
  readonly page: Page;
  readonly welcomeMessage: Locator;
  readonly statsCards: Locator;
  readonly openIssuesCount: Locator;
  readonly inProgressCount: Locator;
  readonly completedCount: Locator;
  readonly projectsList: Locator;
  readonly activeCycleCard: Locator;
  readonly sidebar: Locator;
  readonly topbar: Locator;
  readonly userMenu: Locator;
  readonly createIssueButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.welcomeMessage = page.locator('h1:has-text("Welcome back")');
    this.statsCards = page.locator('.card').filter({ has: page.locator('.text-2xl') });
    this.openIssuesCount = page.locator('.card:has-text("Open Issues") .text-2xl');
    this.inProgressCount = page.locator('.card:has-text("In Progress") .text-2xl');
    this.completedCount = page.locator('.card:has-text("Completed") .text-2xl');
    this.projectsList = page.locator('h2:has-text("Projects")').locator('..').locator('a.card');
    this.activeCycleCard = page.locator('h2:has-text("Active Cycle")').locator('..').locator('.card');
    this.sidebar = page.locator('.sidebar');
    this.topbar = page.locator('.topbar');
    this.userMenu = page.locator('.topbar .avatar');
    this.createIssueButton = page.locator('button:has-text("New Issue")');
  }

  async goto(): Promise<void> {
    await this.page.goto('/app');
  }

  async gotoWorkspace(slug: string): Promise<void> {
    await this.page.goto(`/${slug}`);
  }

  async expectToBeLoggedIn(): Promise<void> {
    await expect(this.sidebar).toBeVisible();
    await expect(this.topbar).toBeVisible();
  }

  async expectWelcomeMessage(name?: string): Promise<void> {
    await expect(this.welcomeMessage).toBeVisible();
    if (name) {
      await expect(this.welcomeMessage).toContainText(name);
    }
  }

  async getOpenIssuesCount(): Promise<number> {
    const text = await this.openIssuesCount.textContent();
    return parseInt(text || '0', 10);
  }

  async getInProgressCount(): Promise<number> {
    const text = await this.inProgressCount.textContent();
    return parseInt(text || '0', 10);
  }

  async getCompletedCount(): Promise<number> {
    const text = await this.completedCount.textContent();
    return parseInt(text || '0', 10);
  }

  async clickProject(name: string): Promise<void> {
    await this.projectsList.filter({ hasText: name }).click();
  }

  async openUserMenu(): Promise<void> {
    // Wait for page to be fully loaded before clicking
    await this.page.waitForLoadState('networkidle');
    await this.userMenu.click();
  }

  async logout(): Promise<void> {
    await this.openUserMenu();
    // Wait for dropdown to open with timeout
    const signOutBtn = this.page.locator('button:has-text("Sign out")');
    await signOutBtn.waitFor({ state: 'visible', timeout: 5000 });
    await signOutBtn.click();
  }

  async expectProjectsVisible(): Promise<void> {
    await expect(this.page.locator('h2:has-text("Projects")')).toBeVisible();
  }
}

import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class DashboardPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto(URLS.dashboard);
  }

  async expectDashboard() {
    await expect(this.page).toHaveURL(/\/wp-admin\/?$/);
    await expect(this.page.locator(SELECTORS.adminMenu)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectWidget(name: string) {
    await expect(this.page.locator(`text=${name}`)).toBeVisible();
  }

  async expectAtAGlanceWidget() {
    await expect(this.page.locator('.at-a-glance, #dashboard_right_now, [data-testid="at-a-glance"]')).toBeVisible();
  }

  async expectActivityWidget() {
    await expect(this.page.locator('.activity, #dashboard_activity, [data-testid="activity"]')).toBeVisible();
  }

  async expectQuickDraftWidget() {
    await expect(this.page.locator('.quick-draft, #dashboard_quick_press, [data-testid="quick-draft"]')).toBeVisible();
  }

  async expectSidebarMenu() {
    await expect(this.page.locator(SELECTORS.adminMenu)).toBeVisible();
  }

  async expectMenuItem(text: string) {
    await expect(this.page.locator(`${SELECTORS.adminMenu} >> text=${text}`)).toBeVisible();
  }

  async clickMenuItem(text: string) {
    await this.page.click(`${SELECTORS.adminMenu} >> text=${text}`);
  }

  async getPostCount(): Promise<string | null> {
    const element = this.page.locator('.post-count, [data-testid="post-count"]');
    return element.textContent();
  }

  async getPageCount(): Promise<string | null> {
    const element = this.page.locator('.page-count, [data-testid="page-count"]');
    return element.textContent();
  }

  async getCommentCount(): Promise<string | null> {
    const element = this.page.locator('.comment-count, [data-testid="comment-count"]');
    return element.textContent();
  }
}

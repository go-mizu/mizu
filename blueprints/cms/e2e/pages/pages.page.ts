import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class PagesPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.pages);
  }

  async gotoNew() {
    await this.page.goto(URLS.pagesNew);
  }

  async gotoEdit(id: string) {
    await this.page.goto(URLS.pageEdit(id));
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.pages);
  }

  // List page assertions
  async expectListPage() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectPageInList(title: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${title}`)).toBeVisible();
  }

  async expectPageNotInList(title: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${title}`)).not.toBeVisible();
  }

  // Status tabs
  async clickStatusTab(status: string) {
    await this.page.click(`${SELECTORS.statusTabs} >> text=${status}`);
  }

  // Search
  async search(query: string) {
    await this.page.fill(SELECTORS.searchBox, query);
    await this.page.click(SELECTORS.searchSubmit);
  }

  // Row actions
  async clickRowAction(pageTitle: string, action: string) {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${pageTitle}")`);
    await row.hover();
    await row.locator(`text=${action}`).click();
  }

  // Edit page
  async expectEditPage() {
    await expect(this.page.locator(SELECTORS.titleInput)).toBeVisible();
  }

  async fillTitle(title: string) {
    await this.page.fill(SELECTORS.titleInput, title);
  }

  async fillContent(content: string) {
    await this.page.fill(SELECTORS.contentEditor, content);
  }

  async selectParentPage(parentTitle: string) {
    await this.page.selectOption('#parent_id, select[name="parent_id"]', { label: parentTitle });
  }

  async setMenuOrder(order: number) {
    await this.page.fill('#menu_order, input[name="menu_order"]', order.toString());
  }

  async publish() {
    await this.page.click(SELECTORS.submitButton);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }

  // Create new page
  async createPage(title: string, content: string, parentTitle?: string) {
    await this.gotoNew();
    await this.fillTitle(title);
    await this.fillContent(content);
    if (parentTitle) {
      await this.selectParentPage(parentTitle);
    }
    await this.publish();
  }
}

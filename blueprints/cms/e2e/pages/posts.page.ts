import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class PostsPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.posts);
  }

  async gotoNew() {
    await this.page.goto(URLS.postsNew);
  }

  async gotoEdit(id: string) {
    await this.page.goto(URLS.postEdit(id));
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.posts);
  }

  // List page assertions
  async expectListPage() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectPostInList(title: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${title}`)).toBeVisible();
  }

  async expectPostNotInList(title: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${title}`)).not.toBeVisible();
  }

  // Status tabs
  async clickStatusTab(status: string) {
    await this.page.click(`${SELECTORS.statusTabs} >> text=${status}`);
  }

  async expectStatusTabActive(status: string) {
    await expect(this.page.locator(`${SELECTORS.statusTabs} .current >> text=${status}`)).toBeVisible();
  }

  // Search
  async search(query: string) {
    await this.page.fill(SELECTORS.searchBox, query);
    await this.page.click(SELECTORS.searchSubmit);
  }

  // Pagination
  async goToNextPage() {
    await this.page.click(SELECTORS.paginationNext);
  }

  async goToPrevPage() {
    await this.page.click(SELECTORS.paginationPrev);
  }

  async expectPagination() {
    await expect(this.page.locator(SELECTORS.pagination)).toBeVisible();
  }

  // Bulk actions
  async selectPost(index: number) {
    await this.page.locator(`${SELECTORS.tableRow}:nth-child(${index + 1}) ${SELECTORS.tableCheckbox}`).check();
  }

  async selectAllPosts() {
    await this.page.locator('thead ' + SELECTORS.tableCheckbox).check();
  }

  async applyBulkAction(action: string) {
    await this.page.selectOption(SELECTORS.bulkActions, action);
    await this.page.click(SELECTORS.bulkApply);
  }

  // Row actions
  async clickRowAction(postTitle: string, action: string) {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${postTitle}")`);
    await row.hover();
    await row.locator(`text=${action}`).click();
  }

  // Post count
  async getPostCount(): Promise<number> {
    const rows = await this.page.locator(SELECTORS.tableRow).count();
    return rows;
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

  async selectCategory(categoryName: string) {
    await this.page.locator(`${SELECTORS.categoryCheckboxes}:has-text("${categoryName}")`).check();
  }

  async addTag(tagName: string) {
    await this.page.fill(SELECTORS.tagInput, tagName);
    await this.page.keyboard.press('Enter');
  }

  async setStatus(status: string) {
    await this.page.selectOption(SELECTORS.statusSelect, status);
  }

  async publish() {
    await this.page.click(SELECTORS.submitButton);
  }

  async saveDraft() {
    await this.page.selectOption(SELECTORS.statusSelect, 'draft');
    await this.page.click(SELECTORS.submitButton);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }

  // Create new post
  async createPost(title: string, content: string, status: string = 'published') {
    await this.gotoNew();
    await this.fillTitle(title);
    await this.fillContent(content);
    if (status !== 'published') {
      await this.setStatus(status);
    }
    await this.publish();
  }
}

import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class TagsPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.tags);
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.tags);
  }

  // List page assertions
  async expectListPage() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectTagInList(name: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${name}`)).toBeVisible();
  }

  async expectTagNotInList(name: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${name}`)).not.toBeVisible();
  }

  // Create tag
  async fillName(name: string) {
    await this.page.fill('#tag-name, input[name="tag-name"]', name);
  }

  async fillSlug(slug: string) {
    await this.page.fill('#tag-slug, input[name="slug"]', slug);
  }

  async fillDescription(description: string) {
    await this.page.fill('#tag-description, textarea[name="description"]', description);
  }

  async submitNewTag() {
    await this.page.click('#submit, input[value="Add New Tag"]');
  }

  async createTag(name: string, slug?: string, description?: string) {
    await this.fillName(name);
    if (slug) await this.fillSlug(slug);
    if (description) await this.fillDescription(description);
    await this.submitNewTag();
  }

  // Row actions
  async clickRowAction(tagName: string, action: string) {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${tagName}")`);
    await row.hover();
    await row.locator(`text=${action}`).click();
  }

  async editTag(tagName: string) {
    await this.clickRowAction(tagName, 'Edit');
  }

  async deleteTag(tagName: string) {
    await this.clickRowAction(tagName, 'Delete');
  }

  async quickEditTag(tagName: string) {
    await this.clickRowAction(tagName, 'Quick Edit');
  }

  // Bulk actions
  async selectTag(index: number) {
    await this.page.locator(`${SELECTORS.tableRow}:nth-child(${index + 1}) ${SELECTORS.tableCheckbox}`).check();
  }

  async selectAllTags() {
    await this.page.locator('thead ' + SELECTORS.tableCheckbox).check();
  }

  async applyBulkAction(action: string) {
    await this.page.selectOption(SELECTORS.bulkActions, action);
    await this.page.click(SELECTORS.bulkApply);
  }

  // Search
  async search(query: string) {
    await this.page.fill(SELECTORS.searchBox, query);
    await this.page.click(SELECTORS.searchSubmit);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }

  // Popular tags
  async expectPopularTags() {
    await expect(this.page.locator('.popular-tags, [data-testid="popular-tags"]')).toBeVisible();
  }
}

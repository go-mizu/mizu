import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class CategoriesPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.categories);
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.categories);
  }

  // List page assertions
  async expectListPage() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectCategoryInList(name: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${name}`)).toBeVisible();
  }

  async expectCategoryNotInList(name: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${name}`)).not.toBeVisible();
  }

  // Create category
  async fillName(name: string) {
    await this.page.fill('#tag-name, input[name="tag-name"]', name);
  }

  async fillSlug(slug: string) {
    await this.page.fill('#tag-slug, input[name="slug"]', slug);
  }

  async fillDescription(description: string) {
    await this.page.fill('#tag-description, textarea[name="description"]', description);
  }

  async selectParent(parentName: string) {
    await this.page.selectOption('#parent, select[name="parent"]', { label: parentName });
  }

  async submitNewCategory() {
    await this.page.click('#submit, input[value="Add New Category"]');
  }

  async createCategory(name: string, slug?: string, description?: string, parentName?: string) {
    await this.fillName(name);
    if (slug) await this.fillSlug(slug);
    if (description) await this.fillDescription(description);
    if (parentName) await this.selectParent(parentName);
    await this.submitNewCategory();
  }

  // Row actions
  async clickRowAction(categoryName: string, action: string) {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${categoryName}")`);
    await row.hover();
    await row.locator(`text=${action}`).click();
  }

  async editCategory(categoryName: string) {
    await this.clickRowAction(categoryName, 'Edit');
  }

  async deleteCategory(categoryName: string) {
    await this.clickRowAction(categoryName, 'Delete');
  }

  async quickEditCategory(categoryName: string) {
    await this.clickRowAction(categoryName, 'Quick Edit');
  }

  // Bulk actions
  async selectCategory(index: number) {
    await this.page.locator(`${SELECTORS.tableRow}:nth-child(${index + 1}) ${SELECTORS.tableCheckbox}`).check();
  }

  async selectAllCategories() {
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

  // Post count
  async getPostCount(categoryName: string): Promise<string | null> {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${categoryName}")`);
    const countCell = row.locator('.posts, td:has-text("Posts")');
    return countCell.textContent();
  }
}

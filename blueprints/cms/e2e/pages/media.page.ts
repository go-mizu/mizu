import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class MediaPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.media);
  }

  async gotoNew() {
    await this.page.goto(URLS.mediaNew);
  }

  async gotoEdit(id: string) {
    await this.page.goto(URLS.mediaEdit(id));
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.media);
  }

  // List page assertions
  async expectListPage() {
    const grid = this.page.locator(SELECTORS.mediaGrid);
    const list = this.page.locator(SELECTORS.table);
    await expect(grid.or(list).first()).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  // View modes
  async switchToGridView() {
    await this.page.click('.view-grid, [data-testid="grid-view"]');
  }

  async switchToListView() {
    await this.page.click('.view-list, [data-testid="list-view"]');
  }

  async expectGridView() {
    await expect(this.page.locator(SELECTORS.mediaGrid)).toBeVisible();
  }

  async expectListView() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  // Filters
  async filterByType(type: string) {
    await this.page.selectOption('.media-type-filter, select[name="attachment-filter"]', type);
  }

  async filterByDate(date: string) {
    await this.page.selectOption('.date-filter, select[name="m"]', date);
  }

  // Search
  async search(query: string) {
    await this.page.fill(SELECTORS.searchBox, query);
    await this.page.click(SELECTORS.searchSubmit);
  }

  // Media item interactions
  async clickMediaItem(index: number) {
    await this.page.locator('.media-item, .attachment').nth(index).click();
  }

  async expectMediaItem(filename: string) {
    await expect(this.page.locator(`text=${filename}`)).toBeVisible();
  }

  // Upload
  async expectUploadPage() {
    await expect(this.page.locator(SELECTORS.uploadDropzone).or(this.page.locator('input[type="file"]'))).toBeVisible();
  }

  async uploadFile(filePath: string) {
    const fileInput = this.page.locator('input[type="file"]');
    await fileInput.setInputFiles(filePath);
  }

  // Edit media
  async expectEditPage() {
    await expect(this.page.locator('#attachment_alt, [name="alt"]')).toBeVisible();
  }

  async fillAltText(altText: string) {
    await this.page.fill('#attachment_alt, [name="alt"]', altText);
  }

  async fillCaption(caption: string) {
    await this.page.fill('#attachment_caption, [name="caption"]', caption);
  }

  async fillDescription(description: string) {
    await this.page.fill('#attachment_description, [name="description"]', description);
  }

  async save() {
    await this.page.click(SELECTORS.submitButton);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }

  async deleteMedia() {
    await this.page.click('.delete-attachment, [data-testid="delete-media"]');
  }
}

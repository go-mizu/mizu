import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class CommentsPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.comments);
  }

  async gotoEdit(id: string) {
    await this.page.goto(URLS.commentEdit(id));
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.comments);
  }

  // List page assertions
  async expectListPage() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectCommentInList(content: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${content}`)).toBeVisible();
  }

  async expectCommentNotInList(content: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${content}`)).not.toBeVisible();
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

  // Row actions
  async clickRowAction(commentContent: string, action: string) {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${commentContent}")`);
    await row.hover();
    await row.locator(`text=${action}`).click();
  }

  async approveComment(commentContent: string) {
    await this.clickRowAction(commentContent, 'Approve');
  }

  async markAsSpam(commentContent: string) {
    await this.clickRowAction(commentContent, 'Spam');
  }

  async trashComment(commentContent: string) {
    await this.clickRowAction(commentContent, 'Trash');
  }

  async replyToComment(commentContent: string, reply: string) {
    await this.clickRowAction(commentContent, 'Reply');
    await this.page.fill('.reply-content, textarea[name="replycontent"]', reply);
    await this.page.click('.reply-submit, input[value="Reply"]');
  }

  // Bulk actions
  async selectComment(index: number) {
    await this.page.locator(`${SELECTORS.tableRow}:nth-child(${index + 1}) ${SELECTORS.tableCheckbox}`).check();
  }

  async selectAllComments() {
    await this.page.locator('thead ' + SELECTORS.tableCheckbox).check();
  }

  async applyBulkAction(action: string) {
    await this.page.selectOption(SELECTORS.bulkActions, action);
    await this.page.click(SELECTORS.bulkApply);
  }

  // Edit page
  async expectEditPage() {
    await expect(this.page.locator('#content, textarea[name="content"]')).toBeVisible();
  }

  async fillContent(content: string) {
    await this.page.fill('#content, textarea[name="content"]', content);
  }

  async setStatus(status: string) {
    await this.page.selectOption('#comment_status, select[name="comment_status"]', status);
  }

  async save() {
    await this.page.click(SELECTORS.submitButton);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }
}

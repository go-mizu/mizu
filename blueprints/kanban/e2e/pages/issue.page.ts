import { type Page, type Locator, expect } from '@playwright/test';

export class IssuePage {
  readonly page: Page;
  readonly issueKey: Locator;
  readonly issueTitle: Locator;
  readonly description: Locator;
  readonly statusSelect: Locator;
  readonly prioritySelect: Locator;
  readonly cycleSelect: Locator;
  readonly assignees: Locator;
  readonly addAssigneeButton: Locator;
  readonly commentInput: Locator;
  readonly commentButton: Locator;
  readonly comments: Locator;
  readonly deleteButton: Locator;
  readonly backButton: Locator;
  readonly projectInfo: Locator;
  readonly createdAt: Locator;

  constructor(page: Page) {
    this.page = page;
    this.issueKey = page.locator('.issue-key').first();
    this.issueTitle = page.locator('#issue-title, h1[contenteditable]');
    this.description = page.locator('#issue-description, .prose[contenteditable]');
    this.statusSelect = page.locator('#issue-status');
    this.prioritySelect = page.locator('#issue-priority');
    this.cycleSelect = page.locator('#issue-cycle');
    this.assignees = page.locator('.card:has-text("Assignees")');
    this.addAssigneeButton = page.locator('[data-modal="assign-modal"]');
    this.commentInput = page.locator('textarea[name="body"]');
    this.commentButton = page.locator('button:has-text("Comment")');
    this.comments = page.locator('.card:has-text("Activity")').locator('.flex.gap-3').filter({ hasNot: page.locator('textarea') });
    this.deleteButton = page.locator('button:has-text("Delete")');
    this.backButton = page.locator('a.btn-ghost.btn-icon').first();
    this.projectInfo = page.locator('.card:has-text("Project")');
    this.createdAt = page.locator('.card:has-text("Created")');
  }

  async goto(workspace: string, key: string): Promise<void> {
    await this.page.goto(`/${workspace}/issue/${key}`);
  }

  async expectToBeOnIssuePage(): Promise<void> {
    await expect(this.issueKey).toBeVisible();
    await expect(this.issueTitle).toBeVisible();
  }

  async getIssueKey(): Promise<string> {
    return await this.issueKey.textContent() || '';
  }

  async getTitle(): Promise<string> {
    return await this.issueTitle.textContent() || '';
  }

  async updateTitle(newTitle: string): Promise<void> {
    await this.issueTitle.click();
    await this.issueTitle.fill(newTitle);
    await this.issueTitle.blur();
  }

  async getStatus(): Promise<string> {
    return await this.statusSelect.inputValue();
  }

  async changeStatus(status: string): Promise<void> {
    await this.statusSelect.selectOption({ label: status });
  }

  async getPriority(): Promise<string> {
    return await this.prioritySelect.inputValue();
  }

  async changePriority(priority: string): Promise<void> {
    await this.prioritySelect.selectOption({ label: priority });
  }

  async addComment(text: string): Promise<void> {
    await this.commentInput.fill(text);
    await this.commentButton.click();
  }

  async expectCommentVisible(text: string): Promise<void> {
    await expect(this.page.locator('.card:has-text("Activity")').locator(`text="${text}"`)).toBeVisible();
  }

  async getCommentCount(): Promise<number> {
    return await this.comments.count();
  }

  async goBack(): Promise<void> {
    await this.backButton.click();
  }

  async deleteIssue(): Promise<void> {
    // Open dropdown menu
    await this.page.locator('.dropdown button').first().click();
    await this.deleteButton.click();
  }

  async expectAssignee(name: string): Promise<void> {
    await expect(this.assignees.locator(`text="${name}"`)).toBeVisible();
  }

  async expectNoAssignees(): Promise<void> {
    await expect(this.assignees.locator('text="No assignees"')).toBeVisible();
  }
}

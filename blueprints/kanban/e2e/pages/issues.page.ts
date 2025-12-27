import { type Page, type Locator, expect } from '@playwright/test';

export class IssuesPage {
  readonly page: Page;
  readonly title: Locator;
  readonly issueCount: Locator;
  readonly issuesTable: Locator;
  readonly issueRows: Locator;
  readonly createIssueButton: Locator;
  readonly createIssueModal: Locator;
  readonly filterButton: Locator;
  readonly emptyState: Locator;

  constructor(page: Page) {
    this.page = page;
    this.title = page.locator('h1:has-text("Issues")');
    this.issueCount = page.locator('p.text-muted:has-text("issues")');
    this.issuesTable = page.locator('.table');
    this.issueRows = page.locator('.table tbody tr');
    this.createIssueButton = page.locator('[data-modal="create-issue-modal"]');
    this.createIssueModal = page.locator('#create-issue-modal');
    this.filterButton = page.locator('button:has-text("Filter")');
    this.emptyState = page.locator('.empty-state');
  }

  async goto(workspace: string): Promise<void> {
    await this.page.goto(`/w/${workspace}/issues`);
  }

  async expectToBeOnIssuesPage(): Promise<void> {
    await expect(this.title).toBeVisible();
  }

  async getIssueCount(): Promise<number> {
    const text = await this.issueCount.textContent();
    const match = text?.match(/(\d+)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  async getIssueRow(key: string): Locator {
    return this.issueRows.filter({ has: this.page.locator(`.issue-key:has-text("${key}")`) });
  }

  async clickIssue(key: string): Promise<void> {
    const row = await this.getIssueRow(key);
    await row.click();
  }

  async openCreateIssueModal(): Promise<void> {
    await this.createIssueButton.click();
    await expect(this.createIssueModal).toBeVisible();
  }

  async createIssue(title: string, description?: string): Promise<void> {
    await this.openCreateIssueModal();
    await this.createIssueModal.locator('#issue-title').fill(title);
    if (description) {
      await this.createIssueModal.locator('#issue-description').fill(description);
    }
    await this.createIssueModal.locator('button:has-text("Create Issue")').click();
  }

  async expectIssueVisible(key: string): Promise<void> {
    const row = await this.getIssueRow(key);
    await expect(row).toBeVisible();
  }

  async expectEmptyState(): Promise<void> {
    await expect(this.emptyState).toBeVisible();
  }

  async getIssueStatus(key: string): Promise<string> {
    const row = await this.getIssueRow(key);
    const status = row.locator('.status-badge');
    return await status.textContent() || '';
  }
}

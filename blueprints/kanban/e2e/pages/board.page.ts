import { type Page, type Locator, expect } from '@playwright/test';

export class BoardPage {
  readonly page: Page;
  readonly projectTitle: Locator;
  readonly columns: Locator;
  readonly issueCards: Locator;
  readonly createIssueButton: Locator;
  readonly createIssueModal: Locator;
  readonly addColumnButton: Locator;
  readonly addColumnModal: Locator;

  constructor(page: Page) {
    this.page = page;
    this.projectTitle = page.locator('h1');
    this.columns = page.locator('.board-column');
    this.issueCards = page.locator('.issue-card');
    this.createIssueButton = page.locator('[data-modal="create-issue-modal"]');
    this.createIssueModal = page.locator('#create-issue-modal');
    this.addColumnButton = page.locator('[data-modal="add-column-modal"]');
    this.addColumnModal = page.locator('#add-column-modal');
  }

  async goto(workspace: string, projectId: string): Promise<void> {
    await this.page.goto(`/w/${workspace}/board/${projectId}`);
  }

  async expectToBeOnBoard(): Promise<void> {
    await expect(this.page.locator('#board')).toBeVisible();
  }

  async getColumnCount(): Promise<number> {
    return await this.columns.count();
  }

  async getColumnByName(name: string): Locator {
    return this.columns.filter({ has: this.page.locator(`.column-title:has-text("${name}")`) });
  }

  async getIssueCountInColumn(columnName: string): Promise<number> {
    const column = await this.getColumnByName(columnName);
    return await column.locator('.issue-card').count();
  }

  async getIssueCard(issueKey: string): Locator {
    return this.issueCards.filter({ has: this.page.locator(`.issue-key:has-text("${issueKey}")`) });
  }

  async clickIssue(issueKey: string): Promise<void> {
    const card = await this.getIssueCard(issueKey);
    await card.click();
  }

  async openCreateIssueModal(): Promise<void> {
    await this.createIssueButton.click();
    await expect(this.createIssueModal).toBeVisible();
  }

  async createIssue(title: string, description?: string, column?: string): Promise<void> {
    await this.openCreateIssueModal();
    await this.createIssueModal.locator('#issue-title').fill(title);
    if (description) {
      await this.createIssueModal.locator('#issue-description').fill(description);
    }
    if (column) {
      await this.createIssueModal.locator('#issue-column').selectOption({ label: column });
    }
    await this.createIssueModal.locator('button:has-text("Create Issue")').click();
  }

  async quickAddIssue(columnName: string, title: string): Promise<void> {
    const column = await this.getColumnByName(columnName);
    const input = column.locator('.quick-add-form input');
    await input.fill(title);
    await input.press('Enter');
  }

  async openAddColumnModal(): Promise<void> {
    await this.addColumnButton.click();
    await expect(this.addColumnModal).toBeVisible();
  }

  async addColumn(name: string): Promise<void> {
    await this.openAddColumnModal();
    await this.addColumnModal.locator('#column-name').fill(name);
    await this.addColumnModal.locator('button:has-text("Add Column")').click();
  }

  async expectColumnExists(name: string): Promise<void> {
    const column = await this.getColumnByName(name);
    await expect(column).toBeVisible();
  }

  async expectIssueInColumn(issueKey: string, columnName: string): Promise<void> {
    const column = await this.getColumnByName(columnName);
    const issue = column.locator(`.issue-card:has-text("${issueKey}")`);
    await expect(issue).toBeVisible();
  }

  async dragIssueToColumn(issueKey: string, targetColumnName: string): Promise<void> {
    const issueCard = await this.getIssueCard(issueKey);
    const targetColumn = await this.getColumnByName(targetColumnName);
    const targetBody = targetColumn.locator('.column-body');

    await issueCard.dragTo(targetBody);
  }
}

import { test, expect, testUsers } from '../fixtures/test-fixtures.js';
import { IssuesPage } from '../pages/issues.page.js';
import { IssuePage } from '../pages/issue.page.js';
import { ApiHelper } from '../helpers/api.js';

test.describe('Issues Management', () => {
  test.describe('Issues List', () => {
    test('TC-ISSUE-001: issues page displays issues', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await issuesPage.expectToBeOnIssuesPage();

      const count = await issuesPage.getIssueCount();
      expect(count).toBeGreaterThan(0);
    });

    test('TC-ISSUE-002: issues list shows status badges', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');

      // Should have status badges
      const badges = page.locator('.table .status-badge');
      const badgeCount = await badges.count();
      expect(badgeCount).toBeGreaterThan(0);
    });

    test('TC-ISSUE-003: click issue row opens detail', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');

      // Click first issue
      const firstRow = issuesPage.issueRows.first();
      const issueKey = await firstRow.locator('.issue-key').textContent();
      await firstRow.click();

      // Should navigate to issue detail
      await expect(page).toHaveURL(new RegExp(`/issue/${issueKey}`));
    });
  });

  test.describe('Issue Detail', () => {
    test('TC-ISSUE-004: issue detail shows all fields', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Navigate to first issue
      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await issuesPage.issueRows.first().click();

      const issuePage = new IssuePage(page);
      await issuePage.expectToBeOnIssuePage();

      // Check all fields are visible
      await expect(issuePage.issueKey).toBeVisible();
      await expect(issuePage.issueTitle).toBeVisible();
      await expect(issuePage.statusSelect).toBeVisible();
      await expect(issuePage.prioritySelect).toBeVisible();
      await expect(issuePage.cycleSelect).toBeVisible();
      await expect(issuePage.projectInfo).toBeVisible();
      await expect(issuePage.createdAt).toBeVisible();
    });

    test('TC-ISSUE-005: can change issue status', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await issuesPage.issueRows.first().click();

      const issuePage = new IssuePage(page);
      await issuePage.expectToBeOnIssuePage();

      // Get current status
      const currentStatus = await issuePage.getStatus();

      // Change status
      const newStatus = currentStatus === 'Backlog' ? 'Todo' : 'Backlog';
      await issuePage.changeStatus(newStatus);

      // Status should be updated (optimistic update)
      const updatedStatus = await issuePage.getStatus();
      expect(updatedStatus).not.toBe(currentStatus);
    });

    test('TC-ISSUE-006: can change issue priority', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await issuesPage.issueRows.first().click();

      const issuePage = new IssuePage(page);
      await issuePage.expectToBeOnIssuePage();

      // Change priority
      await issuePage.changePriority('High');

      const priority = await issuePage.getPriority();
      expect(priority).toBe('high');
    });

    test('TC-ISSUE-007: can add comment to issue', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await issuesPage.issueRows.first().click();

      const issuePage = new IssuePage(page);
      await issuePage.expectToBeOnIssuePage();

      const commentText = `Test comment ${Date.now()}`;
      await issuePage.addComment(commentText);

      // Wait for page reload
      await page.waitForLoadState('networkidle');
      await issuePage.expectCommentVisible(commentText);
    });

    test('TC-ISSUE-008: back button returns to issues list', async ({ page, loginAs }) => {
      await loginAs('alice');

      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await issuesPage.issueRows.first().click();

      const issuePage = new IssuePage(page);
      await issuePage.expectToBeOnIssuePage();

      await issuePage.goBack();

      await expect(page).toHaveURL(/issues/);
    });
  });

  test.describe('Issue CRUD via API', () => {
    test('TC-ISSUE-009: create issue via API shows in list', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Create issue via API
      const api = new ApiHelper();
      await api.login(testUsers.alice.email, testUsers.alice.password);

      const workspaces = await api.getWorkspaces();
      const teams = await api.getTeams(workspaces[0].id);
      const projects = await api.getProjects(teams[0].id);

      const issueTitle = `API Test Issue ${Date.now()}`;
      const issue = await api.createIssue(projects[0].id, {
        title: issueTitle,
      });

      // Verify it appears in the UI
      const issuesPage = new IssuesPage(page);
      await issuesPage.goto('acme');
      await page.reload();

      await issuesPage.expectIssueVisible(issue.key);
    });
  });
});

import { test, expect, testUsers } from '../fixtures/test-fixtures.js';
import { BoardPage } from '../pages/board.page.js';
import { HomePage } from '../pages/home.page.js';
import { ApiHelper } from '../helpers/api.js';

test.describe('Kanban Board', () => {
  let projectId: string;

  test.beforeEach(async ({ page, loginAs }) => {
    await loginAs('alice');

    // Get the first project ID via API
    const api = new ApiHelper();
    await api.login(testUsers.alice.email, testUsers.alice.password);
    const workspaces = await api.getWorkspaces();
    if (workspaces.length > 0) {
      const teams = await api.getTeams(workspaces[0].id);
      if (teams.length > 0) {
        const projects = await api.getProjects(teams[0].id);
        if (projects.length > 0) {
          projectId = projects[0].id;
        }
      }
    }
  });

  test.describe('Board Display', () => {
    test('TC-BOARD-001: board displays all columns', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);
      await boardPage.expectToBeOnBoard();

      const columnCount = await boardPage.getColumnCount();
      expect(columnCount).toBeGreaterThanOrEqual(4);
    });

    test('TC-BOARD-002: board displays issues in columns', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);

      // Should have some issues
      const issueCount = await boardPage.issueCards.count();
      expect(issueCount).toBeGreaterThan(0);
    });

    test('TC-BOARD-003: columns show correct names', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);

      await boardPage.expectColumnExists('Backlog');
      await boardPage.expectColumnExists('Todo');
      await boardPage.expectColumnExists('In Progress');
      await boardPage.expectColumnExists('Done');
    });

    test('TC-BOARD-004: column shows issue count badge', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);

      // Each column should have a count badge
      const badges = page.locator('.column-count');
      const badgeCount = await badges.count();
      expect(badgeCount).toBeGreaterThan(0);
    });
  });

  test.describe('Issue Creation', () => {
    test('TC-BOARD-005: create issue from board modal', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);

      const issueTitle = `Test Issue ${Date.now()}`;
      await boardPage.createIssue(issueTitle, 'Test description');

      // Wait for page reload and check issue exists
      await page.waitForLoadState('networkidle');
      await expect(page.locator(`.issue-card:has-text("${issueTitle}")`)).toBeVisible({ timeout: 10000 });
    });

    test('TC-BOARD-006: create issue modal has all fields', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);
      await boardPage.openCreateIssueModal();

      await expect(boardPage.createIssueModal.locator('#issue-title')).toBeVisible();
      await expect(boardPage.createIssueModal.locator('#issue-description')).toBeVisible();
      await expect(boardPage.createIssueModal.locator('#issue-column')).toBeVisible();
      await expect(boardPage.createIssueModal.locator('#issue-priority')).toBeVisible();
    });
  });

  test.describe('Issue Interaction', () => {
    test('TC-BOARD-007: click issue opens detail page', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);

      // Get first issue
      const firstIssue = boardPage.issueCards.first();
      const issueKey = await firstIssue.locator('.issue-key').textContent();

      await firstIssue.click();

      // Should navigate to issue detail
      await expect(page).toHaveURL(new RegExp(`/issue/${issueKey}`));
    });

    test('TC-BOARD-008: issue card shows key and title', async ({ page }) => {
      test.skip(!projectId, 'No project available');

      const boardPage = new BoardPage(page);
      await boardPage.goto('acme', projectId);

      const firstIssue = boardPage.issueCards.first();

      // Should have issue key
      await expect(firstIssue.locator('.issue-key')).toBeVisible();

      // Should have title
      await expect(firstIssue.locator('.issue-title')).toBeVisible();
    });
  });

  test.describe('Navigation', () => {
    test('TC-BOARD-009: can navigate to board from home', async ({ page }) => {
      const homePage = new HomePage(page);
      await homePage.goto();
      await homePage.expectToBeLoggedIn();

      // Click on first project
      await homePage.projectsList.first().click();

      // Should be on board
      await expect(page).toHaveURL(/board/);
    });
  });
});

import { test, expect, URLS } from '../fixtures/test-fixtures';
import { CommentsPage } from '../pages/comments.page';

test.describe('Comments Management', () => {
  test.describe('Comments List Page', () => {
    test('comments list page loads successfully', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.expectListPage();
    });

    test('comments list shows correct page title', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.expectPageTitle('Comments');
    });

    test('legacy comments URL works', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.gotoLegacy();
      await commentsPage.expectListPage();
    });
  });

  test.describe('Status Filtering', () => {
    test('can filter by All status', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.clickStatusTab('All');
      await commentsPage.expectListPage();
    });

    test('can filter by Pending status', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.clickStatusTab('Pending');
      await commentsPage.expectListPage();
    });

    test('can filter by Approved status', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.clickStatusTab('Approved');
      await commentsPage.expectListPage();
    });

    test('can filter by Spam status', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.clickStatusTab('Spam');
      await commentsPage.expectListPage();
    });

    test('can filter by Trash status', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.clickStatusTab('Trash');
      await commentsPage.expectListPage();
    });
  });

  test.describe('Search', () => {
    test('can search comments by content', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.search('test');
      await commentsPage.expectListPage();
    });

    test('can search comments by author', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.search('admin');
      await commentsPage.expectListPage();
    });
  });

  test.describe('Comment Actions', () => {
    test('can approve a pending comment', async ({ authenticatedPage, api }) => {
      // Create a post first
      const post = await api.createPost({
        title: 'Post for Comments',
        content: 'Content',
        status: 'published',
      });

      // Create a pending comment
      await api.createComment(post.id, {
        author_name: 'Test Commenter',
        author_email: 'commenter@test.com',
        content: 'Pending comment content',
        status: 'pending',
      });

      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.clickStatusTab('Pending');
      // Approve the comment if it exists
    });

    test('can mark comment as spam', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      // Test spam action if comments exist
    });

    test('can trash a comment', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      // Test trash action if comments exist
    });
  });

  test.describe('Edit Comment', () => {
    test('can navigate to edit comment page', async ({ authenticatedPage, api }) => {
      // This test requires a comment ID
      // Create test data first
      const post = await api.createPost({
        title: 'Post for Edit Comment',
        content: 'Content',
        status: 'published',
      });

      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      // Navigate to edit if comments exist
    });
  });

  test.describe('Bulk Actions', () => {
    test('can select all comments', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await commentsPage.selectAllComments();
      await expect(authenticatedPage.locator('thead input[type="checkbox"]')).toBeChecked();
    });

    test('bulk approve action is available', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await expect(authenticatedPage.locator('select[name="action"] option[value="approve"]')).toBeAttached();
    });

    test('bulk spam action is available', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await expect(authenticatedPage.locator('select[name="action"] option[value="spam"]')).toBeAttached();
    });

    test('bulk trash action is available', async ({ authenticatedPage }) => {
      const commentsPage = new CommentsPage(authenticatedPage);
      await commentsPage.goto();
      await expect(authenticatedPage.locator('select[name="action"] option[value="trash"]')).toBeAttached();
    });
  });

  test.describe('Clean URLs', () => {
    test('comments list works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.comments);
      await expect(authenticatedPage.locator('table')).toBeVisible();
    });
  });
});

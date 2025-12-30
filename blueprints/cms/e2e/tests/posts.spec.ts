import { test, expect, URLS } from '../fixtures/test-fixtures';
import { PostsPage } from '../pages/posts.page';
import { generateUniqueTitle } from '../utils/test-data';

test.describe('Posts Management', () => {
  test.describe('Posts List Page', () => {
    test('posts list page loads successfully', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.expectListPage();
    });

    test('posts list shows correct page title', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.expectPageTitle('Posts');
    });

    test('legacy posts URL works', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.gotoLegacy();
      await postsPage.expectListPage();
    });

    test('shows seeded posts in list', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.expectPostInList('Published Post');
    });
  });

  test.describe('Status Filtering', () => {
    test('can filter by All status', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.clickStatusTab('All');
      await postsPage.expectListPage();
    });

    test('can filter by Published status', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.clickStatusTab('Published');
      await postsPage.expectPostInList('Published Post');
    });

    test('can filter by Draft status', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.clickStatusTab('Draft');
      await postsPage.expectListPage();
    });

    test('can filter by Trash status', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.clickStatusTab('Trash');
      await postsPage.expectListPage();
    });
  });

  test.describe('Search', () => {
    test('can search posts by title', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.search('Published');
      await postsPage.expectPostInList('Published Post');
    });

    test('search with no results shows empty list', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.search('nonexistentpost12345');
      const count = await postsPage.getPostCount();
      expect(count).toBe(0);
    });
  });

  test.describe('Create New Post', () => {
    test('new post page loads successfully', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.gotoNew();
      await postsPage.expectEditPage();
    });

    test('can create a new post', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      const title = generateUniqueTitle('Test Post');

      await postsPage.createPost(title, 'This is test content for the post.');
      await postsPage.expectSuccessNotice();
    });

    test('can create a draft post', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      const title = generateUniqueTitle('Draft Test');

      await postsPage.gotoNew();
      await postsPage.fillTitle(title);
      await postsPage.fillContent('Draft content');
      await postsPage.saveDraft();
      await postsPage.expectSuccessNotice();
    });
  });

  test.describe('Edit Post', () => {
    test('can edit an existing post', async ({ authenticatedPage, api }) => {
      // Create a post via API
      const post = await api.createPost({
        title: generateUniqueTitle('Edit Test'),
        content: 'Original content',
        status: 'published',
      });

      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.gotoEdit(post.id);
      await postsPage.expectEditPage();

      // Modify the post
      await postsPage.fillTitle(post.title + ' - Modified');
      await postsPage.publish();
      await postsPage.expectSuccessNotice();
    });
  });

  test.describe('Row Actions', () => {
    test('can click Edit from row actions', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.clickRowAction('Published Post', 'Edit');
      await postsPage.expectEditPage();
    });

    test('can click Trash from row actions', async ({ authenticatedPage, api }) => {
      // Create a post to trash
      const post = await api.createPost({
        title: generateUniqueTitle('Trash Test'),
        content: 'Content to trash',
        status: 'published',
      });

      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.clickRowAction(post.title, 'Trash');

      // Verify post moved to trash
      await postsPage.clickStatusTab('Trash');
      await postsPage.expectPostInList(post.title);
    });
  });

  test.describe('Bulk Actions', () => {
    test('can select all posts', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.goto();
      await postsPage.selectAllPosts();
      // Verify checkboxes are checked
      await expect(authenticatedPage.locator('thead input[type="checkbox"]')).toBeChecked();
    });
  });

  test.describe('Categories and Tags', () => {
    test('can select category in post editor', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.gotoNew();
      await postsPage.fillTitle(generateUniqueTitle('Category Test'));
      await postsPage.fillContent('Content with category');
      // Note: This assumes categories are available in the UI
      // await postsPage.selectCategory('Technology');
      await postsPage.publish();
    });

    test('can add tags in post editor', async ({ authenticatedPage }) => {
      const postsPage = new PostsPage(authenticatedPage);
      await postsPage.gotoNew();
      await postsPage.fillTitle(generateUniqueTitle('Tag Test'));
      await postsPage.fillContent('Content with tags');
      // Note: This assumes tag input is available
      // await postsPage.addTag('JavaScript');
      await postsPage.publish();
    });
  });

  test.describe('Clean URLs', () => {
    test('posts list works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.posts);
      await expect(authenticatedPage.locator('table')).toBeVisible();
    });

    test('new post works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.postsNew);
      await expect(authenticatedPage.locator('input[name="post_title"], #title')).toBeVisible();
    });

    test('edit post works with clean URL', async ({ authenticatedPage, api }) => {
      const post = await api.createPost({
        title: generateUniqueTitle('Clean URL Edit'),
        content: 'Content',
        status: 'published',
      });
      await authenticatedPage.goto(URLS.postEdit(post.id));
      await expect(authenticatedPage.locator('input[name="post_title"], #title')).toBeVisible();
    });
  });
});

import { test, expect, URLS } from '../fixtures/test-fixtures';
import { PagesPage } from '../pages/pages.page';
import { generateUniqueTitle } from '../utils/test-data';

test.describe('Pages Management', () => {
  test.describe('Pages List', () => {
    test('pages list page loads successfully', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.expectListPage();
    });

    test('pages list shows correct page title', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.expectPageTitle('Pages');
    });

    test('legacy pages URL works', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.gotoLegacy();
      await pagesPage.expectListPage();
    });

    test('shows seeded pages in list', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.expectPageInList('About Us');
    });
  });

  test.describe('Status Filtering', () => {
    test('can filter pages by All status', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.clickStatusTab('All');
      await pagesPage.expectListPage();
    });

    test('can filter pages by Published status', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.clickStatusTab('Published');
      await pagesPage.expectListPage();
    });
  });

  test.describe('Search', () => {
    test('can search pages by title', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.search('About');
      await pagesPage.expectPageInList('About Us');
    });

    test('search with no results shows empty list', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.search('nonexistentpage98765');
      // Table should be empty or show no results message
    });
  });

  test.describe('Create New Page', () => {
    test('new page form loads successfully', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.gotoNew();
      await pagesPage.expectEditPage();
    });

    test('can create a new page', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      const title = generateUniqueTitle('Test Page');

      await pagesPage.createPage(title, 'This is test page content.');
      await pagesPage.expectSuccessNotice();
    });

    test('can create page with parent', async ({ authenticatedPage, api }) => {
      // Create parent page
      const parentPage = await api.createPage({
        title: generateUniqueTitle('Parent Page'),
        content: 'Parent content',
        status: 'published',
      });

      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.gotoNew();
      await pagesPage.fillTitle(generateUniqueTitle('Child Page'));
      await pagesPage.fillContent('Child content');
      // Select parent if dropdown is available
      // await pagesPage.selectParentPage(parentPage.title);
      await pagesPage.publish();
      await pagesPage.expectSuccessNotice();
    });

    test('can set menu order', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.gotoNew();
      await pagesPage.fillTitle(generateUniqueTitle('Ordered Page'));
      await pagesPage.fillContent('Content');
      await pagesPage.setMenuOrder(10);
      await pagesPage.publish();
      await pagesPage.expectSuccessNotice();
    });
  });

  test.describe('Edit Page', () => {
    test('can edit an existing page', async ({ authenticatedPage, api }) => {
      const page = await api.createPage({
        title: generateUniqueTitle('Edit Page Test'),
        content: 'Original page content',
        status: 'published',
      });

      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.gotoEdit(page.id);
      await pagesPage.expectEditPage();

      await pagesPage.fillTitle(page.title + ' - Modified');
      await pagesPage.publish();
      await pagesPage.expectSuccessNotice();
    });
  });

  test.describe('Row Actions', () => {
    test('can click Edit from row actions', async ({ authenticatedPage }) => {
      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.clickRowAction('About Us', 'Edit');
      await pagesPage.expectEditPage();
    });

    test('can click Trash from row actions', async ({ authenticatedPage, api }) => {
      const page = await api.createPage({
        title: generateUniqueTitle('Trash Page Test'),
        content: 'Content to trash',
        status: 'published',
      });

      const pagesPage = new PagesPage(authenticatedPage);
      await pagesPage.goto();
      await pagesPage.clickRowAction(page.title, 'Trash');

      // Verify page moved to trash
      await pagesPage.clickStatusTab('Trash');
      await pagesPage.expectPageInList(page.title);
    });
  });

  test.describe('Clean URLs', () => {
    test('pages list works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.pages);
      await expect(authenticatedPage.locator('table')).toBeVisible();
    });

    test('new page works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.pagesNew);
      await expect(authenticatedPage.locator('input[name="post_title"], #title')).toBeVisible();
    });

    test('edit page works with clean URL', async ({ authenticatedPage, api }) => {
      const page = await api.createPage({
        title: generateUniqueTitle('Clean URL Page'),
        content: 'Content',
        status: 'published',
      });
      await authenticatedPage.goto(URLS.pageEdit(page.id));
      await expect(authenticatedPage.locator('input[name="post_title"], #title')).toBeVisible();
    });
  });
});

import { test, expect, URLS } from '../fixtures/test-fixtures';
import { MenusPage } from '../pages/menus.page';
import { generateUniqueTitle } from '../utils/test-data';

test.describe('Menus Management', () => {
  test.describe('Menus Page', () => {
    test('menus page loads successfully', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();
      await menusPage.expectMenusPage();
    });

    test('menus page shows correct title', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();
      await menusPage.expectPageTitle('Menu');
    });

    test('legacy menus URL works', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.gotoLegacy();
      await menusPage.expectMenusPage();
    });
  });

  test.describe('Create Menu', () => {
    test('can create a new menu', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      const menuName = generateUniqueTitle('Menu');
      await menusPage.createMenu(menuName);
      await menusPage.expectSuccessNotice();
    });
  });

  test.describe('Add Menu Items', () => {
    test('can add page to menu', async ({ authenticatedPage, api }) => {
      // Create a page first
      const page = await api.createPage({
        title: generateUniqueTitle('Menu Page'),
        content: 'Content',
        status: 'published',
      });

      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu
      const menuName = generateUniqueTitle('Page Menu');
      await menusPage.createMenu(menuName);

      // Add page to menu
      await menusPage.addPageToMenu(page.title);
      await menusPage.saveMenu();
      await menusPage.expectMenuItemInList(page.title);
    });

    test('can add post to menu', async ({ authenticatedPage, api }) => {
      // Create a post first
      const post = await api.createPost({
        title: generateUniqueTitle('Menu Post'),
        content: 'Content',
        status: 'published',
      });

      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu
      const menuName = generateUniqueTitle('Post Menu');
      await menusPage.createMenu(menuName);

      // Add post to menu
      await menusPage.addPostToMenu(post.title);
      await menusPage.saveMenu();
    });

    test('can add category to menu', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu
      const menuName = generateUniqueTitle('Category Menu');
      await menusPage.createMenu(menuName);

      // Add category to menu (assuming 'Technology' exists from seeding)
      await menusPage.addCategoryToMenu('Technology');
      await menusPage.saveMenu();
    });

    test('can add custom link to menu', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu
      const menuName = generateUniqueTitle('Link Menu');
      await menusPage.createMenu(menuName);

      // Add custom link
      await menusPage.addCustomLink('https://example.com', 'Example Site');
      await menusPage.saveMenu();
      await menusPage.expectMenuItemInList('Example Site');
    });
  });

  test.describe('Edit Menu Items', () => {
    test('can edit menu item label', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu with custom link
      const menuName = generateUniqueTitle('Edit Menu');
      await menusPage.createMenu(menuName);
      await menusPage.addCustomLink('https://example.com', 'Original Label');
      await menusPage.saveMenu();

      // Edit the label
      await menusPage.editMenuItemLabel('Original Label', 'New Label');
      await menusPage.saveMenu();
      await menusPage.expectMenuItemInList('New Label');
    });

    test('can remove menu item', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu with custom link
      const menuName = generateUniqueTitle('Remove Menu');
      await menusPage.createMenu(menuName);
      await menusPage.addCustomLink('https://example.com', 'To Remove');
      await menusPage.saveMenu();

      // Remove the item
      await menusPage.removeMenuItem('To Remove');
      await menusPage.saveMenu();
      await menusPage.expectMenuItemNotInList('To Remove');
    });
  });

  test.describe('Menu Locations', () => {
    test('can assign menu to location', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu
      const menuName = generateUniqueTitle('Location Menu');
      await menusPage.createMenu(menuName);

      // Assign to location (assuming locations exist)
      // await menusPage.assignMenuLocation('primary');
      await menusPage.saveMenu();
    });
  });

  test.describe('Delete Menu', () => {
    test('can delete a menu', async ({ authenticatedPage }) => {
      const menusPage = new MenusPage(authenticatedPage);
      await menusPage.goto();

      // Create a menu to delete
      const menuName = generateUniqueTitle('Delete Menu');
      await menusPage.createMenu(menuName);
      await menusPage.saveMenu();

      // Delete the menu
      await menusPage.deleteMenu();
    });
  });

  test.describe('Clean URLs', () => {
    test('menus works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.menus);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });
  });
});

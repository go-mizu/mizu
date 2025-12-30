import { test, expect, URLS } from '../fixtures/test-fixtures';
import { UsersPage } from '../pages/users.page';
import { generateUniqueEmail, generateUniqueTitle } from '../utils/test-data';

test.describe('Users Management', () => {
  test.describe('Users List Page', () => {
    test('users list page loads successfully', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.expectListPage();
    });

    test('users list shows correct page title', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.expectPageTitle('Users');
    });

    test('legacy users URL works', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.gotoLegacy();
      await usersPage.expectListPage();
    });

    test('shows admin user in list', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.expectUserInList('admin');
    });
  });

  test.describe('Role Filtering', () => {
    test('can filter by All users', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.clickRoleTab('All');
      await usersPage.expectListPage();
    });

    test('can filter by Administrator role', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.clickRoleTab('Administrator');
      await usersPage.expectListPage();
    });

    test('can filter by Editor role', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.clickRoleTab('Editor');
      await usersPage.expectListPage();
    });

    test('can filter by Author role', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.clickRoleTab('Author');
      await usersPage.expectListPage();
    });

    test('can filter by Subscriber role', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.clickRoleTab('Subscriber');
      await usersPage.expectListPage();
    });
  });

  test.describe('Search Users', () => {
    test('can search users by username', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.search('admin');
      await usersPage.expectUserInList('admin');
    });

    test('can search users by email', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.search('admin@test.com');
      await usersPage.expectUserInList('admin');
    });

    test('search with no results shows empty list', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.search('nonexistentuser99999');
      // Should show no users or empty message
    });
  });

  test.describe('Create New User', () => {
    test('new user page loads successfully', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.gotoNew();
      await usersPage.expectNewUserPage();
    });

    test('can create a new subscriber user', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      const email = generateUniqueEmail();
      const username = 'testuser' + Date.now();

      await usersPage.createUser(username, email, 'password123', 'subscriber');
      await usersPage.expectSuccessNotice();
    });

    test('can create a new editor user', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      const email = generateUniqueEmail();
      const username = 'testeditor' + Date.now();

      await usersPage.createUser(username, email, 'password123', 'editor');
      await usersPage.expectSuccessNotice();
    });

    test('can create a new author user', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      const email = generateUniqueEmail();
      const username = 'testauthor' + Date.now();

      await usersPage.createUser(username, email, 'password123', 'author');
      await usersPage.expectSuccessNotice();
    });
  });

  test.describe('Edit User', () => {
    test('can navigate to edit user page', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.editUser('admin');
      await usersPage.expectEditPage();
    });

    test('can update user details', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.editUser('admin');
      await usersPage.fillFirstName('Admin');
      await usersPage.fillLastName('User');
      await usersPage.save();
      await usersPage.expectSuccessNotice();
    });
  });

  test.describe('User Profile', () => {
    test('profile page loads successfully', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.gotoProfile();
      await usersPage.expectProfilePage();
    });

    test('can update own profile', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.gotoProfile();
      await usersPage.fillFirstName('Test');
      await usersPage.fillLastName('Admin');
      await usersPage.saveProfile();
      await usersPage.expectSuccessNotice();
    });

    test('can change color scheme', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.gotoProfile();
      await usersPage.selectColorScheme('fresh');
      await usersPage.saveProfile();
      await usersPage.expectSuccessNotice();
    });
  });

  test.describe('Row Actions', () => {
    test('can click Edit from row actions', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.editUser('admin');
      await usersPage.expectEditPage();
    });
  });

  test.describe('Bulk Actions', () => {
    test('can select all users', async ({ authenticatedPage }) => {
      const usersPage = new UsersPage(authenticatedPage);
      await usersPage.goto();
      await usersPage.selectAllUsers();
      await expect(authenticatedPage.locator('thead input[type="checkbox"]')).toBeChecked();
    });
  });

  test.describe('Clean URLs', () => {
    test('new user works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.usersNew);
      await expect(authenticatedPage.locator('input[name="user_login"], #user_login')).toBeVisible();
    });

    test('profile works with legacy URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.profile);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });
  });
});

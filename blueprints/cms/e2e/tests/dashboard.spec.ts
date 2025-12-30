import { test, expect, URLS } from '../fixtures/test-fixtures';
import { DashboardPage } from '../pages/dashboard.page';

test.describe('Dashboard', () => {
  test.describe('Page Rendering', () => {
    test('dashboard page loads successfully', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectDashboard();
    });

    test('dashboard shows correct page title', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectPageTitle('Dashboard');
    });

    test('legacy dashboard URL works', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.legacy.dashboard);
      await expect(authenticatedPage).toHaveURL(/\/wp-admin\//);
    });
  });

  test.describe('Widgets', () => {
    test('At a Glance widget is visible', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectAtAGlanceWidget();
    });

    test('Activity widget is visible', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectActivityWidget();
    });

    test('Quick Draft widget is visible', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectQuickDraftWidget();
    });
  });

  test.describe('Navigation Sidebar', () => {
    test('sidebar menu is visible', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectSidebarMenu();
    });

    test('sidebar contains Posts menu item', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectMenuItem('Posts');
    });

    test('sidebar contains Pages menu item', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectMenuItem('Pages');
    });

    test('sidebar contains Media menu item', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectMenuItem('Media');
    });

    test('sidebar contains Comments menu item', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectMenuItem('Comments');
    });

    test('sidebar contains Users menu item', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectMenuItem('Users');
    });

    test('sidebar contains Settings menu item', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.expectMenuItem('Settings');
    });

    test('clicking Posts navigates to posts list', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.clickMenuItem('Posts');
      await expect(authenticatedPage).toHaveURL(/posts|edit\.php/);
    });

    test('clicking Pages navigates to pages list', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      await dashboard.clickMenuItem('Pages');
      await expect(authenticatedPage).toHaveURL(/pages|post_type=page/);
    });
  });

  test.describe('Stats Display', () => {
    test('shows post count', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      // Just verify the element exists - actual count depends on test data
      const count = await dashboard.getPostCount();
      expect(count).not.toBeNull();
    });

    test('shows page count', async ({ authenticatedPage }) => {
      const dashboard = new DashboardPage(authenticatedPage);
      await dashboard.goto();
      const count = await dashboard.getPageCount();
      expect(count).not.toBeNull();
    });
  });
});

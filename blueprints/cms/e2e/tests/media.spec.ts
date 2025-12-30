import { test, expect, URLS } from '../fixtures/test-fixtures';
import { MediaPage } from '../pages/media.page';

test.describe('Media Library', () => {
  test.describe('Media List Page', () => {
    test('media library page loads successfully', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.expectListPage();
    });

    test('media library shows correct page title', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.expectPageTitle('Media');
    });

    test('legacy media URL works', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.gotoLegacy();
      await mediaPage.expectListPage();
    });
  });

  test.describe('View Modes', () => {
    test('can switch to grid view', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.switchToGridView();
      // Grid view should be visible
    });

    test('can switch to list view', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.switchToListView();
      // List view should be visible
    });
  });

  test.describe('Filters', () => {
    test('can filter by media type - images', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.filterByType('image');
      await mediaPage.expectListPage();
    });

    test('can filter by media type - videos', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.filterByType('video');
      await mediaPage.expectListPage();
    });

    test('can filter by media type - audio', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.filterByType('audio');
      await mediaPage.expectListPage();
    });
  });

  test.describe('Search', () => {
    test('can search media by filename', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.goto();
      await mediaPage.search('test');
      await mediaPage.expectListPage();
    });
  });

  test.describe('Upload Page', () => {
    test('upload page loads successfully', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.gotoNew();
      await mediaPage.expectUploadPage();
    });

    test('upload page shows correct title', async ({ authenticatedPage }) => {
      const mediaPage = new MediaPage(authenticatedPage);
      await mediaPage.gotoNew();
      await mediaPage.expectPageTitle('Upload');
    });
  });

  test.describe('Clean URLs', () => {
    test('media library works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.media);
      await expect(authenticatedPage.locator('.media-grid, table')).toBeVisible();
    });

    test('new media works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.mediaNew);
      await expect(authenticatedPage.locator('input[type="file"], .upload-dropzone')).toBeVisible();
    });
  });
});

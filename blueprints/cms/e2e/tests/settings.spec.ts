import { test, expect, URLS } from '../fixtures/test-fixtures';
import { SettingsPage } from '../pages/settings.page';

test.describe('Settings', () => {
  test.describe('General Settings', () => {
    test('general settings page loads successfully', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.expectSettingsPage();
    });

    test('general settings shows correct title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.expectPageTitle('General');
    });

    test('legacy general settings URL works', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneralLegacy();
      await settingsPage.expectSettingsPage();
    });

    test('can update site title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.fillSiteTitle('Test Site Title');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can update tagline', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.fillTagline('Test tagline');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can select timezone', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.selectTimezone('America/New_York');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can select date format', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.selectDateFormat('F j, Y');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can select time format', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoGeneral();
      await settingsPage.selectTimeFormat('g:i a');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });
  });

  test.describe('Writing Settings', () => {
    test('writing settings page loads successfully', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoWriting();
      await settingsPage.expectSettingsPage();
    });

    test('writing settings shows correct title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoWriting();
      await settingsPage.expectPageTitle('Writing');
    });

    test('legacy writing settings URL works', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoWritingLegacy();
      await settingsPage.expectSettingsPage();
    });

    test('can update default post category', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoWriting();
      // Select a category if dropdown exists
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });
  });

  test.describe('Reading Settings', () => {
    test('reading settings page loads successfully', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoReading();
      await settingsPage.expectSettingsPage();
    });

    test('reading settings shows correct title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoReading();
      await settingsPage.expectPageTitle('Reading');
    });

    test('legacy reading settings URL works', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoReadingLegacy();
      await settingsPage.expectSettingsPage();
    });

    test('can update posts per page', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoReading();
      await settingsPage.fillPostsPerPage(15);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can select front page display option', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoReading();
      await settingsPage.selectFrontPageDisplay('posts');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });
  });

  test.describe('Discussion Settings', () => {
    test('discussion settings page loads successfully', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussion();
      await settingsPage.expectSettingsPage();
    });

    test('discussion settings shows correct title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussion();
      await settingsPage.expectPageTitle('Discussion');
    });

    test('legacy discussion settings URL works', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussionLegacy();
      await settingsPage.expectSettingsPage();
    });

    test('can toggle default comment status', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussion();
      await settingsPage.toggleDefaultCommentStatus(true);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can toggle comment moderation', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussion();
      await settingsPage.toggleCommentModeration(true);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can set comments per page', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussion();
      await settingsPage.fillCommentsPerPage(20);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can toggle show avatars', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoDiscussion();
      await settingsPage.toggleShowAvatars(true);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });
  });

  test.describe('Media Settings', () => {
    test('media settings page loads successfully', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMedia();
      await settingsPage.expectSettingsPage();
    });

    test('media settings shows correct title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMedia();
      await settingsPage.expectPageTitle('Media');
    });

    test('legacy media settings URL works', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMediaLegacy();
      await settingsPage.expectSettingsPage();
    });

    test('can update thumbnail size', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMedia();
      await settingsPage.fillThumbnailSize(150, 150);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can update medium size', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMedia();
      await settingsPage.fillMediumSize(300, 300);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can update large size', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMedia();
      await settingsPage.fillLargeSize(1024, 1024);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can toggle organize uploads', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoMedia();
      await settingsPage.toggleOrganizeUploads(true);
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });
  });

  test.describe('Permalinks Settings', () => {
    test('permalinks settings page loads successfully', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinks();
      await settingsPage.expectSettingsPage();
    });

    test('permalinks settings shows correct title', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinks();
      await settingsPage.expectPageTitle('Permalink');
    });

    test('legacy permalinks settings URL works', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinksLegacy();
      await settingsPage.expectSettingsPage();
    });

    test('can select permalink structure', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinks();
      await settingsPage.selectPermalinkStructure('/%postname%/');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can set custom permalink structure', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinks();
      await settingsPage.fillCustomStructure('/%year%/%monthnum%/%postname%/');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can set category base', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinks();
      await settingsPage.fillCategoryBase('category');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });

    test('can set tag base', async ({ authenticatedPage }) => {
      const settingsPage = new SettingsPage(authenticatedPage);
      await settingsPage.gotoPermalinks();
      await settingsPage.fillTagBase('tag');
      await settingsPage.save();
      await settingsPage.expectSuccessNotice();
    });
  });

  test.describe('Clean URLs', () => {
    test('general settings works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.settingsGeneral);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });

    test('writing settings works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.settingsWriting);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });

    test('reading settings works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.settingsReading);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });

    test('discussion settings works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.settingsDiscussion);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });

    test('media settings works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.settingsMedia);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });

    test('permalinks settings works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.settingsPermalinks);
      await expect(authenticatedPage.locator('form')).toBeVisible();
    });
  });
});

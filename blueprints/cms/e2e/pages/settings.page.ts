import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class SettingsPage {
  constructor(private page: Page) {}

  // Navigation - General
  async gotoGeneral() {
    await this.page.goto(URLS.settingsGeneral);
  }

  async gotoGeneralLegacy() {
    await this.page.goto(URLS.legacy.settingsGeneral);
  }

  // Navigation - Writing
  async gotoWriting() {
    await this.page.goto(URLS.settingsWriting);
  }

  async gotoWritingLegacy() {
    await this.page.goto(URLS.legacy.settingsWriting);
  }

  // Navigation - Reading
  async gotoReading() {
    await this.page.goto(URLS.settingsReading);
  }

  async gotoReadingLegacy() {
    await this.page.goto(URLS.legacy.settingsReading);
  }

  // Navigation - Discussion
  async gotoDiscussion() {
    await this.page.goto(URLS.settingsDiscussion);
  }

  async gotoDiscussionLegacy() {
    await this.page.goto(URLS.legacy.settingsDiscussion);
  }

  // Navigation - Media
  async gotoMedia() {
    await this.page.goto(URLS.settingsMedia);
  }

  async gotoMediaLegacy() {
    await this.page.goto(URLS.legacy.settingsMedia);
  }

  // Navigation - Permalinks
  async gotoPermalinks() {
    await this.page.goto(URLS.settingsPermalinks);
  }

  async gotoPermalinksLegacy() {
    await this.page.goto(URLS.legacy.settingsPermalinks);
  }

  // Page assertions
  async expectSettingsPage() {
    await expect(this.page.locator(SELECTORS.settingsForm)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  // Save
  async save() {
    await this.page.click(SELECTORS.settingsSave);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }

  // General settings
  async fillSiteTitle(title: string) {
    await this.page.fill('#blogname, input[name="blogname"]', title);
  }

  async fillTagline(tagline: string) {
    await this.page.fill('#blogdescription, input[name="blogdescription"]', tagline);
  }

  async fillSiteUrl(url: string) {
    await this.page.fill('#siteurl, input[name="siteurl"]', url);
  }

  async fillHomeUrl(url: string) {
    await this.page.fill('#home, input[name="home"]', url);
  }

  async fillAdminEmail(email: string) {
    await this.page.fill('#new_admin_email, input[name="new_admin_email"]', email);
  }

  async selectTimezone(timezone: string) {
    await this.page.selectOption('#timezone_string, select[name="timezone_string"]', timezone);
  }

  async selectDateFormat(format: string) {
    await this.page.click(`input[name="date_format"][value="${format}"]`);
  }

  async fillCustomDateFormat(format: string) {
    await this.page.click('input[name="date_format"][value="\\custom"]');
    await this.page.fill('#date_format_custom, input[name="date_format_custom"]', format);
  }

  async selectTimeFormat(format: string) {
    await this.page.click(`input[name="time_format"][value="${format}"]`);
  }

  async fillCustomTimeFormat(format: string) {
    await this.page.click('input[name="time_format"][value="\\custom"]');
    await this.page.fill('#time_format_custom, input[name="time_format_custom"]', format);
  }

  async toggleMembership(enable: boolean) {
    const checkbox = this.page.locator('#users_can_register, input[name="users_can_register"]');
    if (enable) {
      await checkbox.check();
    } else {
      await checkbox.uncheck();
    }
  }

  // Writing settings
  async selectDefaultCategory(categoryName: string) {
    await this.page.selectOption('#default_category, select[name="default_category"]', { label: categoryName });
  }

  async selectDefaultPostFormat(format: string) {
    await this.page.selectOption('#default_post_format, select[name="default_post_format"]', format);
  }

  async fillUpdateServices(services: string) {
    await this.page.fill('#ping_sites, textarea[name="ping_sites"]', services);
  }

  // Reading settings
  async selectFrontPageDisplay(type: 'posts' | 'page') {
    if (type === 'posts') {
      await this.page.click('#show_on_front[value="posts"]');
    } else {
      await this.page.click('#show_on_front[value="page"]');
    }
  }

  async selectStaticFrontPage(pageTitle: string) {
    await this.page.selectOption('#page_on_front, select[name="page_on_front"]', { label: pageTitle });
  }

  async selectPostsPage(pageTitle: string) {
    await this.page.selectOption('#page_for_posts, select[name="page_for_posts"]', { label: pageTitle });
  }

  async fillPostsPerPage(count: number) {
    await this.page.fill('#posts_per_page, input[name="posts_per_page"]', count.toString());
  }

  async selectRssFeedContent(type: 'full' | 'summary') {
    if (type === 'full') {
      await this.page.click('#rss_use_excerpt[value="0"]');
    } else {
      await this.page.click('#rss_use_excerpt[value="1"]');
    }
  }

  async toggleSearchEngineVisibility(discourage: boolean) {
    const checkbox = this.page.locator('#blog_public, input[name="blog_public"]');
    if (discourage) {
      await checkbox.uncheck();
    } else {
      await checkbox.check();
    }
  }

  // Discussion settings
  async toggleDefaultCommentStatus(enable: boolean) {
    const checkbox = this.page.locator('#default_comment_status, input[name="default_comment_status"]');
    if (enable) {
      await checkbox.check();
    } else {
      await checkbox.uncheck();
    }
  }

  async toggleCommentModeration(enable: boolean) {
    const checkbox = this.page.locator('#comment_moderation, input[name="comment_moderation"]');
    if (enable) {
      await checkbox.check();
    } else {
      await checkbox.uncheck();
    }
  }

  async fillCommentsPerPage(count: number) {
    await this.page.fill('#comments_per_page, input[name="comments_per_page"]', count.toString());
  }

  async selectThreadingDepth(depth: number) {
    await this.page.selectOption('#thread_comments_depth, select[name="thread_comments_depth"]', depth.toString());
  }

  async toggleShowAvatars(show: boolean) {
    const checkbox = this.page.locator('#show_avatars, input[name="show_avatars"]');
    if (show) {
      await checkbox.check();
    } else {
      await checkbox.uncheck();
    }
  }

  // Media settings
  async fillThumbnailSize(width: number, height: number) {
    await this.page.fill('#thumbnail_size_w, input[name="thumbnail_size_w"]', width.toString());
    await this.page.fill('#thumbnail_size_h, input[name="thumbnail_size_h"]', height.toString());
  }

  async fillMediumSize(width: number, height: number) {
    await this.page.fill('#medium_size_w, input[name="medium_size_w"]', width.toString());
    await this.page.fill('#medium_size_h, input[name="medium_size_h"]', height.toString());
  }

  async fillLargeSize(width: number, height: number) {
    await this.page.fill('#large_size_w, input[name="large_size_w"]', width.toString());
    await this.page.fill('#large_size_h, input[name="large_size_h"]', height.toString());
  }

  async toggleOrganizeUploads(organize: boolean) {
    const checkbox = this.page.locator('#uploads_use_yearmonth_folders, input[name="uploads_use_yearmonth_folders"]');
    if (organize) {
      await checkbox.check();
    } else {
      await checkbox.uncheck();
    }
  }

  // Permalinks settings
  async selectPermalinkStructure(structure: string) {
    await this.page.click(`input[name="selection"][value="${structure}"]`);
  }

  async fillCustomStructure(structure: string) {
    await this.page.click('input[name="selection"][value="custom"]');
    await this.page.fill('#permalink_structure, input[name="permalink_structure"]', structure);
  }

  async fillCategoryBase(base: string) {
    await this.page.fill('#category_base, input[name="category_base"]', base);
  }

  async fillTagBase(base: string) {
    await this.page.fill('#tag_base, input[name="tag_base"]', base);
  }
}

import { Page, expect, Locator } from '@playwright/test';

/**
 * Page object for the public-facing site (frontend theme).
 */
export class SitePage {
  readonly page: Page;

  // Common selectors
  readonly header: Locator;
  readonly footer: Locator;
  readonly mainContent: Locator;
  readonly sidebar: Locator;
  readonly navigation: Locator;
  readonly searchForm: Locator;
  readonly themeToggle: Locator;

  constructor(page: Page) {
    this.page = page;
    this.header = page.locator('header, #site-header, .site-header');
    this.footer = page.locator('footer, #site-footer, .site-footer');
    this.mainContent = page.locator('main, #main, .main-content, .site-main');
    this.sidebar = page.locator('aside, #sidebar, .sidebar');
    this.navigation = page.locator('nav, #primary-nav, .primary-nav');
    this.searchForm = page.locator('form[action*="search"], .search-form');
    this.themeToggle = page.locator('#theme-toggle, .theme-toggle');
  }

  // Navigation methods
  async gotoHome(): Promise<void> {
    await this.page.goto('/');
  }

  async gotoPost(slug: string): Promise<void> {
    await this.page.goto(`/${slug}`);
  }

  async gotoPage(slug: string): Promise<void> {
    await this.page.goto(`/page/${slug}`);
  }

  async gotoCategory(slug: string): Promise<void> {
    await this.page.goto(`/category/${slug}`);
  }

  async gotoTag(slug: string): Promise<void> {
    await this.page.goto(`/tag/${slug}`);
  }

  async gotoAuthor(slug: string): Promise<void> {
    await this.page.goto(`/author/${slug}`);
  }

  async gotoArchive(): Promise<void> {
    await this.page.goto('/archive');
  }

  async gotoSearch(query: string = ''): Promise<void> {
    if (query) {
      await this.page.goto(`/search?q=${encodeURIComponent(query)}`);
    } else {
      await this.page.goto('/search');
    }
  }

  async gotoFeed(): Promise<void> {
    await this.page.goto('/feed');
  }

  async goto404(): Promise<void> {
    await this.page.goto('/nonexistent-page-12345');
  }

  // Assertion methods
  async expectHomePage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    // Home page typically has a hero or posts grid
    await expect(
      this.page.locator('.hero, .posts-grid, .post-card, .home-content, article')
    ).toBeVisible();
  }

  async expectPostPage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    // Single post should have article with title
    await expect(
      this.page.locator('article, .single-post, .post-content')
    ).toBeVisible();
    await expect(
      this.page.locator('h1, .post-title, .entry-title')
    ).toBeVisible();
  }

  async expectPageContent(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    await expect(
      this.page.locator('article, .page-content, .entry-content')
    ).toBeVisible();
  }

  async expectCategoryPage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    // Category page should show archive header or category info
    await expect(
      this.page.locator('.archive-header, .category-header, h1')
    ).toBeVisible();
  }

  async expectTagPage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    await expect(
      this.page.locator('.archive-header, .tag-header, h1')
    ).toBeVisible();
  }

  async expectAuthorPage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    await expect(
      this.page.locator('.author-header, .author-info, .archive-header, h1')
    ).toBeVisible();
  }

  async expectArchivePage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    await expect(
      this.page.locator('.archive-header, .posts-list, h1')
    ).toBeVisible();
  }

  async expectSearchPage(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    await expect(
      this.page.locator('.search-header, .search-results, .search-form, h1')
    ).toBeVisible();
  }

  async expect404Page(): Promise<void> {
    await expect(this.header).toBeVisible();
    await expect(this.mainContent).toBeVisible();
    // 404 page should indicate error
    const errorIndicator = this.page.locator(
      '.error-404, .not-found, [class*="404"], h1:has-text("404"), h1:has-text("Not Found")'
    );
    await expect(errorIndicator).toBeVisible();
  }

  async expectRSSFeed(): Promise<void> {
    const content = await this.page.content();
    expect(content).toContain('<?xml');
    expect(content).toContain('<rss');
    expect(content).toContain('<channel>');
  }

  // Content verification methods
  async expectPostInList(title: string): Promise<void> {
    await expect(
      this.page.locator(`article:has-text("${title}"), .post-card:has-text("${title}"), a:has-text("${title}")`)
    ).toBeVisible();
  }

  async expectPostTitle(title: string): Promise<void> {
    await expect(
      this.page.locator(`h1:has-text("${title}"), .post-title:has-text("${title}"), .entry-title:has-text("${title}")`)
    ).toBeVisible();
  }

  async expectPageTitle(title: string): Promise<void> {
    await expect(
      this.page.locator(`h1:has-text("${title}"), .page-title:has-text("${title}")`)
    ).toBeVisible();
  }

  async expectCategoryName(name: string): Promise<void> {
    await expect(
      this.page.locator(`h1:has-text("${name}"), .category-title:has-text("${name}"), .archive-title:has-text("${name}")`)
    ).toBeVisible();
  }

  async expectTagName(name: string): Promise<void> {
    await expect(
      this.page.locator(`h1:has-text("${name}"), .tag-title:has-text("${name}"), .archive-title:has-text("${name}")`)
    ).toBeVisible();
  }

  async expectAuthorName(name: string): Promise<void> {
    await expect(
      this.page.locator(`h1:has-text("${name}"), .author-name:has-text("${name}"), .archive-title:has-text("${name}")`)
    ).toBeVisible();
  }

  // Interaction methods
  async search(query: string): Promise<void> {
    const searchInput = this.page.locator(
      'input[name="q"], input[name="s"], input[type="search"], .search-input'
    );
    await searchInput.fill(query);
    await searchInput.press('Enter');
    await this.page.waitForLoadState('networkidle');
  }

  async clickPost(title: string): Promise<void> {
    await this.page.locator(`a:has-text("${title}")`).first().click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickCategory(name: string): Promise<void> {
    await this.page.locator(`a:has-text("${name}")`).first().click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickTag(name: string): Promise<void> {
    await this.page.locator(`a:has-text("${name}")`).first().click();
    await this.page.waitForLoadState('networkidle');
  }

  async toggleDarkMode(): Promise<void> {
    if (await this.themeToggle.isVisible()) {
      await this.themeToggle.click();
    }
  }

  // Pagination
  async hasNextPage(): Promise<boolean> {
    const nextBtn = this.page.locator(
      '.pagination-next, .next, a:has-text("Next"), a[rel="next"]'
    );
    return nextBtn.isVisible();
  }

  async hasPrevPage(): Promise<boolean> {
    const prevBtn = this.page.locator(
      '.pagination-prev, .prev, a:has-text("Previous"), a[rel="prev"]'
    );
    return prevBtn.isVisible();
  }

  async goToNextPage(): Promise<void> {
    const nextBtn = this.page.locator(
      '.pagination-next, .next, a:has-text("Next"), a[rel="next"]'
    );
    await nextBtn.click();
    await this.page.waitForLoadState('networkidle');
  }

  async goToPrevPage(): Promise<void> {
    const prevBtn = this.page.locator(
      '.pagination-prev, .prev, a:has-text("Previous"), a[rel="prev"]'
    );
    await prevBtn.click();
    await this.page.waitForLoadState('networkidle');
  }

  // Theme verification
  async expectDarkMode(): Promise<void> {
    const html = this.page.locator('html');
    await expect(html).toHaveAttribute('data-theme', 'dark');
  }

  async expectLightMode(): Promise<void> {
    const html = this.page.locator('html');
    await expect(html).toHaveAttribute('data-theme', 'light');
  }

  // SEO checks
  async expectMetaTitle(title: string): Promise<void> {
    const metaTitle = await this.page.title();
    expect(metaTitle).toContain(title);
  }

  async expectMetaDescription(): Promise<void> {
    const metaDesc = this.page.locator('meta[name="description"]');
    await expect(metaDesc).toHaveAttribute('content', /.+/);
  }

  async expectCanonicalURL(): Promise<void> {
    const canonical = this.page.locator('link[rel="canonical"]');
    await expect(canonical).toHaveAttribute('href', /.+/);
  }

  // Responsive checks
  async setMobileViewport(): Promise<void> {
    await this.page.setViewportSize({ width: 375, height: 667 });
  }

  async setTabletViewport(): Promise<void> {
    await this.page.setViewportSize({ width: 768, height: 1024 });
  }

  async setDesktopViewport(): Promise<void> {
    await this.page.setViewportSize({ width: 1280, height: 800 });
  }

  // Get element counts
  async getPostCount(): Promise<number> {
    return this.page.locator('article, .post-card').count();
  }

  async getCommentCount(): Promise<number> {
    return this.page.locator('.comment, .comment-item').count();
  }
}

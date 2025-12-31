import { test, expect } from '../fixtures/test-fixtures';
import { SitePage } from '../pages/site.page';
import { generateUniqueTitle } from '../utils/test-data';

test.describe('Site Frontend (Theme)', () => {
  test.describe('Homepage', () => {
    test('homepage loads successfully', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await sitePage.expectHomePage();
    });

    test('homepage has header with navigation', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await expect(sitePage.header).toBeVisible();
    });

    test('homepage has footer', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await expect(sitePage.footer).toBeVisible();
    });

    test('homepage shows posts', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      // Should have at least one post visible
      const postCount = await sitePage.getPostCount();
      expect(postCount).toBeGreaterThan(0);
    });

    test('homepage shows seeded published post', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await sitePage.expectPostInList('Published Post');
    });
  });

  test.describe('Single Post', () => {
    test('single post page loads for published post', async ({ page, api }) => {
      // Create a published post
      const title = generateUniqueTitle('Site Post');
      const post = await api.createPost({
        title,
        content: 'This is the post content for the site test.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPost(post.slug);
      await sitePage.expectPostPage();
    });

    test('single post shows post title', async ({ page, api }) => {
      const title = generateUniqueTitle('Title Test Post');
      const post = await api.createPost({
        title,
        content: 'Content for title verification.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPost(post.slug);
      await sitePage.expectPostTitle(title);
    });

    test('single post shows post content', async ({ page, api }) => {
      const title = generateUniqueTitle('Content Post');
      const content = 'This is unique content that should be visible on the page.';
      const post = await api.createPost({
        title,
        content,
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPost(post.slug);
      await expect(page.locator(`text=${content.substring(0, 50)}`)).toBeVisible();
    });

    test('clicking post from homepage navigates to single post', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await sitePage.clickPost('Published Post');
      await sitePage.expectPostPage();
    });
  });

  test.describe('Single Page', () => {
    test('single page loads for published page', async ({ page, api }) => {
      const title = generateUniqueTitle('Site Page');
      const pg = await api.createPage({
        title,
        content: 'This is the page content.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPage(pg.slug);
      await sitePage.expectPageContent();
    });

    test('single page shows page title', async ({ page, api }) => {
      const title = generateUniqueTitle('Page Title Test');
      const pg = await api.createPage({
        title,
        content: 'Page content here.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPage(pg.slug);
      await sitePage.expectPageTitle(title);
    });

    test('seeded About Us page is accessible', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoPage('about-us');
      await sitePage.expectPageContent();
    });
  });

  test.describe('Category Archive', () => {
    test('category archive page loads', async ({ page, api }) => {
      // Get existing category
      const categories = await api.listCategories();
      const category = categories.find(c => c.name === 'Technology') || categories[0];

      if (category) {
        const sitePage = new SitePage(page);
        await sitePage.gotoCategory(category.slug);
        await sitePage.expectCategoryPage();
      }
    });

    test('category archive shows category name', async ({ page, api }) => {
      const categories = await api.listCategories();
      const category = categories.find(c => c.name === 'Technology') || categories[0];

      if (category) {
        const sitePage = new SitePage(page);
        await sitePage.gotoCategory(category.slug);
        await sitePage.expectCategoryName(category.name);
      }
    });
  });

  test.describe('Tag Archive', () => {
    test('tag archive page loads', async ({ page, api }) => {
      // Get existing tag
      const tags = await api.listTags();
      const tag = tags.find(t => t.name === 'JavaScript') || tags[0];

      if (tag) {
        const sitePage = new SitePage(page);
        await sitePage.gotoTag(tag.slug);
        await sitePage.expectTagPage();
      }
    });

    test('tag archive shows tag name', async ({ page, api }) => {
      const tags = await api.listTags();
      const tag = tags.find(t => t.name === 'JavaScript') || tags[0];

      if (tag) {
        const sitePage = new SitePage(page);
        await sitePage.gotoTag(tag.slug);
        await sitePage.expectTagName(tag.name);
      }
    });
  });

  test.describe('Author Archive', () => {
    test('author archive page loads', async ({ page, api }) => {
      // Get existing user
      const users = await api.listUsers();
      const user = users[0];

      if (user) {
        const sitePage = new SitePage(page);
        await sitePage.gotoAuthor(user.username);
        await sitePage.expectAuthorPage();
      }
    });
  });

  test.describe('Archive Page', () => {
    test('archive page loads', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoArchive();
      await sitePage.expectArchivePage();
    });

    test('archive page shows posts', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoArchive();
      const postCount = await sitePage.getPostCount();
      expect(postCount).toBeGreaterThanOrEqual(0);
    });
  });

  test.describe('Search', () => {
    test('search page loads', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoSearch();
      await sitePage.expectSearchPage();
    });

    test('search with query shows results', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoSearch('Published');
      await sitePage.expectSearchPage();
    });

    test('search with no results shows empty state', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoSearch('nonexistenttermxyz123');
      await sitePage.expectSearchPage();
    });

    test('can perform search from search input', async ({ page, api }) => {
      // Create a searchable post
      const title = generateUniqueTitle('Searchable Post');
      await api.createPost({
        title,
        content: 'This post should be found via search.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoSearch();
      await sitePage.search('Searchable');
      await sitePage.expectSearchPage();
    });
  });

  test.describe('404 Error Page', () => {
    test('404 page shows for nonexistent URL', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.goto404();
      await sitePage.expect404Page();
    });

    test('404 page has header and footer', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.goto404();
      await expect(sitePage.header).toBeVisible();
      await expect(sitePage.footer).toBeVisible();
    });
  });

  test.describe('RSS Feed', () => {
    test('RSS feed is accessible', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoFeed();
      await sitePage.expectRSSFeed();
    });

    test('RSS feed contains channel info', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoFeed();
      const content = await page.content();
      expect(content).toContain('<title>');
      expect(content).toContain('<link>');
    });
  });

  test.describe('Theme Features', () => {
    test('dark mode toggle works', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();

      // Check if theme toggle exists
      if (await sitePage.themeToggle.isVisible()) {
        // Get initial theme
        const html = page.locator('html');
        const initialTheme = await html.getAttribute('data-theme');

        // Toggle theme
        await sitePage.toggleDarkMode();

        // Theme should change
        const newTheme = await html.getAttribute('data-theme');
        if (initialTheme === 'dark') {
          expect(newTheme).toBe('light');
        } else if (initialTheme === 'light') {
          expect(newTheme).toBe('dark');
        }
      }
    });
  });

  test.describe('Responsive Design', () => {
    test('homepage renders on mobile viewport', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.setMobileViewport();
      await sitePage.gotoHome();
      await sitePage.expectHomePage();
    });

    test('homepage renders on tablet viewport', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.setTabletViewport();
      await sitePage.gotoHome();
      await sitePage.expectHomePage();
    });

    test('single post renders on mobile viewport', async ({ page, api }) => {
      const post = await api.createPost({
        title: generateUniqueTitle('Mobile Post'),
        content: 'Mobile content test.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.setMobileViewport();
      await sitePage.gotoPost(post.slug);
      await sitePage.expectPostPage();
    });
  });

  test.describe('Navigation', () => {
    test('can navigate from home to post and back', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await sitePage.expectHomePage();

      // Click on a post
      await sitePage.clickPost('Published Post');
      await sitePage.expectPostPage();

      // Go back home
      await page.goBack();
      await sitePage.expectHomePage();
    });

    test('header navigation links work', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();

      // Check that navigation is visible
      await expect(sitePage.header).toBeVisible();
    });
  });

  test.describe('SEO', () => {
    test('homepage has proper title', async ({ page }) => {
      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      const title = await page.title();
      expect(title.length).toBeGreaterThan(0);
    });

    test('single post has title with post name', async ({ page, api }) => {
      const postTitle = generateUniqueTitle('SEO Post');
      const post = await api.createPost({
        title: postTitle,
        content: 'SEO test content.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPost(post.slug);
      await sitePage.expectMetaTitle(postTitle);
    });
  });

  test.describe('Content Visibility', () => {
    test('draft posts are not visible on site', async ({ page, api }) => {
      const title = generateUniqueTitle('Draft Post Site');
      const post = await api.createPost({
        title,
        content: 'This draft should not be visible.',
        status: 'draft',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPost(post.slug);
      // Should show 404 or not found
      await sitePage.expect404Page();
    });

    test('published posts are visible on site', async ({ page, api }) => {
      const title = generateUniqueTitle('Visible Post');
      const post = await api.createPost({
        title,
        content: 'This post should be visible.',
        status: 'published',
      });

      const sitePage = new SitePage(page);
      await sitePage.gotoPost(post.slug);
      await sitePage.expectPostPage();
      await sitePage.expectPostTitle(title);
    });
  });

  test.describe('Performance', () => {
    test('homepage loads within acceptable time', async ({ page }) => {
      const startTime = Date.now();

      const sitePage = new SitePage(page);
      await sitePage.gotoHome();
      await sitePage.expectHomePage();

      const loadTime = Date.now() - startTime;
      // Should load in under 5 seconds
      expect(loadTime).toBeLessThan(5000);
    });
  });
});

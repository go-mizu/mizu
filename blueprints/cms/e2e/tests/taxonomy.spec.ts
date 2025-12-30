import { test, expect, URLS } from '../fixtures/test-fixtures';
import { CategoriesPage } from '../pages/categories.page';
import { TagsPage } from '../pages/tags.page';
import { generateUniqueTitle } from '../utils/test-data';

test.describe('Taxonomy Management', () => {
  test.describe('Categories', () => {
    test.describe('Categories List Page', () => {
      test('categories page loads successfully', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.expectListPage();
      });

      test('categories page shows correct title', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.expectPageTitle('Categories');
      });

      test('legacy categories URL works', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.gotoLegacy();
        await categoriesPage.expectListPage();
      });

      test('shows seeded categories in list', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.expectCategoryInList('Technology');
      });
    });

    test.describe('Create Category', () => {
      test('can create a new category', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();

        const name = generateUniqueTitle('Category');
        await categoriesPage.createCategory(name);
        await categoriesPage.expectSuccessNotice();
        await categoriesPage.expectCategoryInList(name);
      });

      test('can create category with description', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();

        const name = generateUniqueTitle('Described Cat');
        await categoriesPage.createCategory(name, undefined, 'This is a description');
        await categoriesPage.expectSuccessNotice();
      });

      test('can create category with custom slug', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();

        const name = generateUniqueTitle('Slugged Cat');
        await categoriesPage.createCategory(name, 'custom-slug-' + Date.now());
        await categoriesPage.expectSuccessNotice();
      });

      test('can create child category', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();

        const childName = generateUniqueTitle('Child Category');
        // Assuming 'Technology' parent exists from seeding
        await categoriesPage.createCategory(childName, undefined, undefined, 'Technology');
        await categoriesPage.expectSuccessNotice();
      });
    });

    test.describe('Category Row Actions', () => {
      test('can click Edit from row actions', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.editCategory('Technology');
        // Should navigate to edit page
      });

      test('can click Quick Edit from row actions', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.quickEditCategory('Technology');
        // Quick edit form should appear
      });
    });

    test.describe('Search Categories', () => {
      test('can search categories by name', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.search('Technology');
        await categoriesPage.expectCategoryInList('Technology');
      });
    });

    test.describe('Bulk Actions', () => {
      test('can select all categories', async ({ authenticatedPage }) => {
        const categoriesPage = new CategoriesPage(authenticatedPage);
        await categoriesPage.goto();
        await categoriesPage.selectAllCategories();
        await expect(authenticatedPage.locator('thead input[type="checkbox"]')).toBeChecked();
      });
    });
  });

  test.describe('Tags', () => {
    test.describe('Tags List Page', () => {
      test('tags page loads successfully', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.expectListPage();
      });

      test('tags page shows correct title', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.expectPageTitle('Tags');
      });

      test('legacy tags URL works', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.gotoLegacy();
        await tagsPage.expectListPage();
      });

      test('shows seeded tags in list', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.expectTagInList('JavaScript');
      });
    });

    test.describe('Create Tag', () => {
      test('can create a new tag', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();

        const name = generateUniqueTitle('Tag');
        await tagsPage.createTag(name);
        await tagsPage.expectSuccessNotice();
        await tagsPage.expectTagInList(name);
      });

      test('can create tag with description', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();

        const name = generateUniqueTitle('Described Tag');
        await tagsPage.createTag(name, undefined, 'Tag description');
        await tagsPage.expectSuccessNotice();
      });

      test('can create tag with custom slug', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();

        const name = generateUniqueTitle('Slugged Tag');
        await tagsPage.createTag(name, 'custom-tag-' + Date.now());
        await tagsPage.expectSuccessNotice();
      });
    });

    test.describe('Tag Row Actions', () => {
      test('can click Edit from row actions', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.editTag('JavaScript');
        // Should navigate to edit page
      });

      test('can click Quick Edit from row actions', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.quickEditTag('JavaScript');
        // Quick edit form should appear
      });
    });

    test.describe('Search Tags', () => {
      test('can search tags by name', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.search('JavaScript');
        await tagsPage.expectTagInList('JavaScript');
      });
    });

    test.describe('Bulk Actions', () => {
      test('can select all tags', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.selectAllTags();
        await expect(authenticatedPage.locator('thead input[type="checkbox"]')).toBeChecked();
      });
    });

    test.describe('Popular Tags', () => {
      test('shows popular tags section', async ({ authenticatedPage }) => {
        const tagsPage = new TagsPage(authenticatedPage);
        await tagsPage.goto();
        await tagsPage.expectPopularTags();
      });
    });
  });

  test.describe('Clean URLs', () => {
    test('categories works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.categories);
      await expect(authenticatedPage.locator('table')).toBeVisible();
    });

    test('tags works with clean URL', async ({ authenticatedPage }) => {
      await authenticatedPage.goto(URLS.tags);
      await expect(authenticatedPage.locator('table')).toBeVisible();
    });
  });
});

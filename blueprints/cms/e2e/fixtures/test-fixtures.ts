import { test as base, expect, Page } from '@playwright/test';
import { APIClient } from '../utils/api-client';
import { defaultTestData } from '../utils/test-data';

// Test credentials
export const TEST_ADMIN = {
  email: defaultTestData.adminUser.email,
  password: defaultTestData.adminUser.password,
  username: defaultTestData.adminUser.username,
};

// Page URLs
export const URLS = {
  // Public site URLs (frontend theme)
  site: {
    home: '/',
    post: (slug: string) => `/${slug}`,
    page: (slug: string) => `/page/${slug}`,
    category: (slug: string) => `/category/${slug}`,
    tag: (slug: string) => `/tag/${slug}`,
    author: (slug: string) => `/author/${slug}`,
    archive: '/archive',
    search: '/search',
    searchWithQuery: (q: string) => `/search?q=${encodeURIComponent(q)}`,
    feed: '/feed',
  },

  // Clean URLs (without .php)
  login: '/wp-admin/login',
  dashboard: '/wp-admin/',
  posts: '/wp-admin/posts',
  postsNew: '/wp-admin/posts/new',
  postEdit: (id: string) => `/wp-admin/posts/${id}`,
  pages: '/wp-admin/pages',
  pagesNew: '/wp-admin/pages/new',
  pageEdit: (id: string) => `/wp-admin/pages/${id}`,
  media: '/wp-admin/media',
  mediaNew: '/wp-admin/media/new',
  mediaEdit: (id: string) => `/wp-admin/media/${id}`,
  comments: '/wp-admin/comments',
  commentEdit: (id: string) => `/wp-admin/comments/${id}`,
  categories: '/wp-admin/categories',
  tags: '/wp-admin/tags',
  menus: '/wp-admin/menus',
  users: '/wp-admin/users.php',
  usersNew: '/wp-admin/users/new',
  userEdit: (id: string) => `/wp-admin/users/${id}`,
  profile: '/wp-admin/profile.php',
  settingsGeneral: '/wp-admin/settings/general',
  settingsWriting: '/wp-admin/settings/writing',
  settingsReading: '/wp-admin/settings/reading',
  settingsDiscussion: '/wp-admin/settings/discussion',
  settingsMedia: '/wp-admin/settings/media',
  settingsPermalinks: '/wp-admin/settings/permalinks',

  // Legacy URLs (with .php) - for backwards compatibility testing
  legacy: {
    login: '/wp-login.php',
    dashboard: '/wp-admin/index.php',
    posts: '/wp-admin/edit.php',
    postsNew: '/wp-admin/post-new.php',
    postEdit: '/wp-admin/post.php',
    pages: '/wp-admin/edit.php?post_type=page',
    pagesNew: '/wp-admin/post-new.php?post_type=page',
    pageEdit: '/wp-admin/post.php?post_type=page',
    media: '/wp-admin/upload.php',
    mediaNew: '/wp-admin/media-new.php',
    comments: '/wp-admin/edit-comments.php',
    commentEdit: '/wp-admin/comment.php',
    categories: '/wp-admin/edit-tags.php?taxonomy=category',
    tags: '/wp-admin/edit-tags.php?taxonomy=post_tag',
    menus: '/wp-admin/nav-menus.php',
    users: '/wp-admin/users.php',
    usersNew: '/wp-admin/user-new.php',
    userEdit: '/wp-admin/user-edit.php',
    profile: '/wp-admin/profile.php',
    settingsGeneral: '/wp-admin/options-general.php',
    settingsWriting: '/wp-admin/options-writing.php',
    settingsReading: '/wp-admin/options-reading.php',
    settingsDiscussion: '/wp-admin/options-discussion.php',
    settingsMedia: '/wp-admin/options-media.php',
    settingsPermalinks: '/wp-admin/options-permalink.php',
  },
};

// Extend base test with custom fixtures
interface TestFixtures {
  api: APIClient;
  authenticatedPage: Page;
}

export const test = base.extend<TestFixtures>({
  api: async ({ request }, use) => {
    const api = new APIClient(request);
    await use(api);
  },

  authenticatedPage: async ({ page, api }, use) => {
    // Login via API to get session
    const { session } = await api.login(TEST_ADMIN.email, TEST_ADMIN.password);

    // Set session cookie
    await page.context().addCookies([
      {
        name: 'session',
        value: session,
        domain: 'localhost',
        path: '/',
      },
    ]);

    await use(page);
  },
});

export { expect };

// Helper function to login via UI
export async function loginViaUI(page: Page, email: string, password: string): Promise<void> {
  await page.goto(URLS.login);
  await page.fill('input[name="log"], input[name="email"], #user_login', email);
  await page.fill('input[name="pwd"], input[name="password"], #user_pass', password);
  await page.click('input[type="submit"], button[type="submit"], #wp-submit');
  await page.waitForURL(/\/wp-admin\//);
}

// Helper to check if user is logged in
export async function isLoggedIn(page: Page): Promise<boolean> {
  const cookies = await page.context().cookies();
  return cookies.some((c) => c.name === 'session' && c.value !== '');
}

// Selectors for common elements
export const SELECTORS = {
  // Admin layout
  adminMenu: '#adminmenu, .admin-menu, [data-testid="admin-menu"]',
  adminBar: '#wpadminbar, .admin-bar, [data-testid="admin-bar"]',
  pageTitle: '.wrap h1, h1.wp-heading-inline, [data-testid="page-title"]',
  notice: '.notice, .updated, .error, [data-testid="notice"]',

  // Login form
  loginForm: '#loginform, form[name="loginform"], [data-testid="login-form"]',
  loginUsername: '#user_login, input[name="log"], input[name="email"]',
  loginPassword: '#user_pass, input[name="pwd"], input[name="password"]',
  loginSubmit: '#wp-submit, input[type="submit"], button[type="submit"]',
  loginError: '#login_error, .login-error, [data-testid="login-error"]',

  // Tables
  table: '.wp-list-table, table.widefat, [data-testid="list-table"]',
  tableRow: 'tbody tr, .wp-list-table tbody tr',
  tableCheckbox: 'input[type="checkbox"].check-column, th.check-column input',

  // Pagination
  pagination: '.tablenav-pages, [data-testid="pagination"]',
  paginationNext: '.next-page, a.next-page',
  paginationPrev: '.prev-page, a.prev-page',

  // Bulk actions
  bulkActions: '#bulk-action-selector-top, select[name="action"]',
  bulkApply: '#doaction, input[value="Apply"]',

  // Search
  searchBox: '#post-search-input, input[name="s"], [data-testid="search-input"]',
  searchSubmit: '#search-submit, input[value="Search"]',

  // Tabs
  statusTabs: '.subsubsub, [data-testid="status-tabs"]',

  // Forms
  titleInput: '#title, input[name="post_title"], [data-testid="title-input"]',
  contentEditor: '#content, textarea[name="content"], [data-testid="content-editor"]',
  submitButton: '#publish, input[name="save"], button[type="submit"]',

  // Post/Page specific
  categoryCheckboxes: '#categorychecklist input[type="checkbox"]',
  tagInput: '#new-tag-post_tag, input[name="newtag"]',
  statusSelect: '#post_status, select[name="post_status"]',
  visibilitySelect: '#visibility, select[name="visibility"]',

  // Media
  mediaGrid: '.media-grid, [data-testid="media-grid"]',
  mediaListView: '.media-list, [data-testid="media-list"]',
  uploadDropzone: '.upload-dropzone, [data-testid="upload-dropzone"]',

  // User specific
  roleSelect: '#role, select[name="role"]',
  emailInput: '#email, input[name="email"]',
  passwordInput: '#pass1, input[name="pass1"]',

  // Menu builder
  menuItemsList: '#menu-to-edit, [data-testid="menu-items"]',
  availableMenuItems: '.accordion-container, [data-testid="available-items"]',

  // Settings
  settingsForm: 'form[action*="options"], [data-testid="settings-form"]',
  settingsSave: '#submit, input[value="Save Changes"]',
};

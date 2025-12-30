import { APIClient } from './api-client';

export interface TestData {
  adminUser: { email: string; password: string; username: string };
  editorUser: { email: string; password: string; username: string };
  posts: Array<{ title: string; content: string; status: string }>;
  pages: Array<{ title: string; content: string; status: string; parent?: string }>;
  categories: Array<{ name: string; description: string; parent?: string }>;
  tags: Array<{ name: string; description: string }>;
  settings: Record<string, string>;
}

export const defaultTestData: TestData = {
  adminUser: {
    email: 'admin@test.com',
    password: 'password123',
    username: 'admin',
  },
  editorUser: {
    email: 'editor@test.com',
    password: 'password123',
    username: 'editor',
  },
  posts: [
    { title: 'Published Post', content: 'This is a published post.', status: 'published' },
    { title: 'Draft Post', content: 'This is a draft post.', status: 'draft' },
    { title: 'Scheduled Post', content: 'This is a scheduled post.', status: 'scheduled' },
    { title: 'Another Published', content: 'Another published post.', status: 'published' },
    { title: 'Trash Post', content: 'This is a trashed post.', status: 'trash' },
  ],
  pages: [
    { title: 'About Us', content: 'About us page content.', status: 'published' },
    { title: 'Contact', content: 'Contact page content.', status: 'published' },
    { title: 'Services', content: 'Services page content.', status: 'published' },
  ],
  categories: [
    { name: 'Technology', description: 'Technology related posts' },
    { name: 'Programming', description: 'Programming posts', parent: 'Technology' },
    { name: 'News', description: 'News posts' },
    { name: 'Tutorials', description: 'Tutorial posts' },
    { name: 'Reviews', description: 'Product reviews' },
  ],
  tags: [
    { name: 'JavaScript', description: 'JavaScript related' },
    { name: 'TypeScript', description: 'TypeScript related' },
    { name: 'Go', description: 'Go language related' },
    { name: 'React', description: 'React framework' },
    { name: 'Testing', description: 'Testing related' },
    { name: 'Tutorial', description: 'Tutorial content' },
    { name: 'Beginner', description: 'Beginner friendly' },
    { name: 'Advanced', description: 'Advanced topics' },
    { name: 'Best Practices', description: 'Best practices' },
    { name: 'Tips', description: 'Tips and tricks' },
  ],
  settings: {
    site_title: 'Test CMS Site',
    site_tagline: 'Just another CMS site',
    admin_email: 'admin@test.com',
    timezone: 'America/New_York',
    date_format: 'F j, Y',
    time_format: 'g:i a',
    posts_per_page: '10',
    permalink_structure: '/%postname%/',
  },
};

export async function seedTestData(api: APIClient, data: TestData = defaultTestData): Promise<void> {
  // Register admin user
  try {
    await api.register(data.adminUser.email, data.adminUser.password, data.adminUser.username);
  } catch {
    // User might already exist, try to login
    await api.login(data.adminUser.email, data.adminUser.password);
  }

  // Create categories
  const categoryMap: Record<string, string> = {};
  for (const cat of data.categories) {
    try {
      const parentId = cat.parent ? categoryMap[cat.parent] : undefined;
      const created = await api.createCategory({
        name: cat.name,
        slug: cat.name.toLowerCase().replace(/\s+/g, '-'),
        description: cat.description,
        parent_id: parentId,
      });
      categoryMap[cat.name] = created.id;
    } catch {
      // Category might already exist
    }
  }

  // Create tags
  for (const tag of data.tags) {
    try {
      await api.createTag({
        name: tag.name,
        slug: tag.name.toLowerCase().replace(/\s+/g, '-'),
        description: tag.description,
      });
    } catch {
      // Tag might already exist
    }
  }

  // Create posts
  for (const post of data.posts) {
    try {
      await api.createPost({
        title: post.title,
        slug: post.title.toLowerCase().replace(/\s+/g, '-'),
        content: post.content,
        status: post.status,
      });
    } catch {
      // Post might already exist
    }
  }

  // Create pages
  for (const page of data.pages) {
    try {
      await api.createPage({
        title: page.title,
        slug: page.title.toLowerCase().replace(/\s+/g, '-'),
        content: page.content,
        status: page.status,
      });
    } catch {
      // Page might already exist
    }
  }

  // Set settings
  try {
    await api.setSettings(data.settings);
  } catch {
    // Settings might fail
  }
}

export function generateUniqueEmail(): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return `test-${timestamp}-${random}@test.com`;
}

export function generateUniqueTitle(prefix: string): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return `${prefix} ${timestamp}-${random}`;
}

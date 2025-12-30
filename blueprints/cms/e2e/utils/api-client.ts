import { APIRequestContext } from '@playwright/test';

export interface User {
  id: string;
  email: string;
  username: string;
  display_name: string;
  role: string;
}

export interface Post {
  id: string;
  title: string;
  slug: string;
  content: string;
  excerpt: string;
  status: string;
  author_id: string;
}

export interface Page {
  id: string;
  title: string;
  slug: string;
  content: string;
  status: string;
  parent_id?: string;
  menu_order: number;
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  description: string;
  parent_id?: string;
}

export interface Tag {
  id: string;
  name: string;
  slug: string;
  description: string;
}

export interface Comment {
  id: string;
  post_id: string;
  author_name: string;
  author_email: string;
  content: string;
  status: string;
}

export interface Menu {
  id: string;
  name: string;
  slug: string;
  location: string;
}

export class APIClient {
  private request: APIRequestContext;
  private baseURL: string;
  private sessionCookie?: string;

  constructor(request: APIRequestContext, baseURL: string = 'http://localhost:8080') {
    this.request = request;
    this.baseURL = baseURL;
  }

  setSession(cookie: string) {
    this.sessionCookie = cookie;
  }

  private get headers() {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.sessionCookie) {
      headers['Cookie'] = `session=${this.sessionCookie}`;
    }
    return headers;
  }

  // Auth
  async register(email: string, password: string, username: string): Promise<User> {
    const response = await this.request.post(`${this.baseURL}/api/v1/auth/register`, {
      headers: this.headers,
      data: { email, password, username, display_name: username },
    });
    const data = await response.json();
    if (data.session) {
      this.sessionCookie = data.session;
    }
    return data.user;
  }

  async login(email: string, password: string): Promise<{ user: User; session: string }> {
    const response = await this.request.post(`${this.baseURL}/api/v1/auth/login`, {
      headers: this.headers,
      data: { email, password },
    });
    const data = await response.json();
    if (data.session) {
      this.sessionCookie = data.session;
    }
    return data;
  }

  // Users
  async listUsers(): Promise<User[]> {
    const response = await this.request.get(`${this.baseURL}/api/v1/users`, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.users || [];
  }

  async getUser(id: string): Promise<User> {
    const response = await this.request.get(`${this.baseURL}/api/v1/users/${id}`, {
      headers: this.headers,
    });
    return response.json();
  }

  async deleteUser(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/users/${id}`, {
      headers: this.headers,
    });
  }

  // Posts
  async createPost(post: Partial<Post>): Promise<Post> {
    const response = await this.request.post(`${this.baseURL}/api/v1/posts`, {
      headers: this.headers,
      data: post,
    });
    return response.json();
  }

  async listPosts(status?: string): Promise<Post[]> {
    const url = status
      ? `${this.baseURL}/api/v1/posts?status=${status}`
      : `${this.baseURL}/api/v1/posts`;
    const response = await this.request.get(url, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.posts || [];
  }

  async getPost(id: string): Promise<Post> {
    const response = await this.request.get(`${this.baseURL}/api/v1/posts/${id}`, {
      headers: this.headers,
    });
    return response.json();
  }

  async updatePost(id: string, post: Partial<Post>): Promise<Post> {
    const response = await this.request.put(`${this.baseURL}/api/v1/posts/${id}`, {
      headers: this.headers,
      data: post,
    });
    return response.json();
  }

  async deletePost(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/posts/${id}`, {
      headers: this.headers,
    });
  }

  // Pages
  async createPage(page: Partial<Page>): Promise<Page> {
    const response = await this.request.post(`${this.baseURL}/api/v1/pages`, {
      headers: this.headers,
      data: page,
    });
    return response.json();
  }

  async listPages(): Promise<Page[]> {
    const response = await this.request.get(`${this.baseURL}/api/v1/pages`, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.pages || [];
  }

  async getPage(id: string): Promise<Page> {
    const response = await this.request.get(`${this.baseURL}/api/v1/pages/${id}`, {
      headers: this.headers,
    });
    return response.json();
  }

  async updatePage(id: string, page: Partial<Page>): Promise<Page> {
    const response = await this.request.put(`${this.baseURL}/api/v1/pages/${id}`, {
      headers: this.headers,
      data: page,
    });
    return response.json();
  }

  async deletePage(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/pages/${id}`, {
      headers: this.headers,
    });
  }

  // Categories
  async createCategory(category: Partial<Category>): Promise<Category> {
    const response = await this.request.post(`${this.baseURL}/api/v1/categories`, {
      headers: this.headers,
      data: category,
    });
    return response.json();
  }

  async listCategories(): Promise<Category[]> {
    const response = await this.request.get(`${this.baseURL}/api/v1/categories`, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.categories || [];
  }

  async getCategory(id: string): Promise<Category> {
    const response = await this.request.get(`${this.baseURL}/api/v1/categories/${id}`, {
      headers: this.headers,
    });
    return response.json();
  }

  async deleteCategory(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/categories/${id}`, {
      headers: this.headers,
    });
  }

  // Tags
  async createTag(tag: Partial<Tag>): Promise<Tag> {
    const response = await this.request.post(`${this.baseURL}/api/v1/tags`, {
      headers: this.headers,
      data: tag,
    });
    return response.json();
  }

  async listTags(): Promise<Tag[]> {
    const response = await this.request.get(`${this.baseURL}/api/v1/tags`, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.tags || [];
  }

  async getTag(id: string): Promise<Tag> {
    const response = await this.request.get(`${this.baseURL}/api/v1/tags/${id}`, {
      headers: this.headers,
    });
    return response.json();
  }

  async deleteTag(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/tags/${id}`, {
      headers: this.headers,
    });
  }

  // Comments
  async createComment(postId: string, comment: Partial<Comment>): Promise<Comment> {
    const response = await this.request.post(`${this.baseURL}/api/v1/comments/for-post/${postId}`, {
      headers: this.headers,
      data: comment,
    });
    return response.json();
  }

  async listComments(status?: string): Promise<Comment[]> {
    const url = status
      ? `${this.baseURL}/api/v1/comments?status=${status}`
      : `${this.baseURL}/api/v1/comments`;
    const response = await this.request.get(url, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.comments || [];
  }

  async approveComment(id: string): Promise<void> {
    await this.request.post(`${this.baseURL}/api/v1/comments/approve/${id}`, {
      headers: this.headers,
    });
  }

  async deleteComment(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/comments/${id}`, {
      headers: this.headers,
    });
  }

  // Menus
  async createMenu(menu: Partial<Menu>): Promise<Menu> {
    const response = await this.request.post(`${this.baseURL}/api/v1/menus`, {
      headers: this.headers,
      data: menu,
    });
    return response.json();
  }

  async listMenus(): Promise<Menu[]> {
    const response = await this.request.get(`${this.baseURL}/api/v1/menus`, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.menus || [];
  }

  async deleteMenu(id: string): Promise<void> {
    await this.request.delete(`${this.baseURL}/api/v1/menus/${id}`, {
      headers: this.headers,
    });
  }

  // Settings
  async getSetting(key: string): Promise<string> {
    const response = await this.request.get(`${this.baseURL}/api/v1/settings/${key}`, {
      headers: this.headers,
    });
    const data = await response.json();
    return data.value;
  }

  async setSetting(key: string, value: string): Promise<void> {
    await this.request.put(`${this.baseURL}/api/v1/settings/${key}`, {
      headers: this.headers,
      data: { value },
    });
  }

  async setSettings(settings: Record<string, string>): Promise<void> {
    await this.request.put(`${this.baseURL}/api/v1/settings`, {
      headers: this.headers,
      data: settings,
    });
  }
}

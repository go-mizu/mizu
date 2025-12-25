import { APIRequestContext } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';

export interface User {
  id: string;
  username: string;
  display_name?: string;
  email?: string;
}

export interface Chat {
  id: string;
  type: 'direct' | 'group';
  group_name?: string;
  other_user?: User;
}

export interface Message {
  id: string;
  chat_id: string;
  sender_id: string;
  content: string;
  type: string;
  status: string;
  created_at: string;
}

export class ApiHelper {
  private request: APIRequestContext;
  private authToken?: string;

  constructor(request: APIRequestContext) {
    this.request = request;
  }

  private get headers(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }
    return headers;
  }

  // Authentication
  async register(data: {
    username: string;
    password: string;
    display_name?: string;
    email?: string;
  }): Promise<{ user: User; token: string }> {
    const response = await this.request.post(`${BASE_URL}/api/v1/auth/register`, {
      headers: this.headers,
      data,
    });
    const result = await response.json();
    if (result.success) {
      this.authToken = result.token;
      return { user: result.user, token: result.token };
    }
    throw new Error(result.error || 'Registration failed');
  }

  async login(login: string, password: string): Promise<{ user: User; token: string }> {
    const response = await this.request.post(`${BASE_URL}/api/v1/auth/login`, {
      headers: this.headers,
      data: { login, password },
    });
    const result = await response.json();
    if (result.success) {
      this.authToken = result.token;
      return { user: result.user, token: result.token };
    }
    throw new Error(result.error || 'Login failed');
  }

  async logout(): Promise<void> {
    await this.request.post(`${BASE_URL}/api/v1/auth/logout`, {
      headers: this.headers,
    });
    this.authToken = undefined;
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.request.get(`${BASE_URL}/api/v1/users/me`, {
      headers: this.headers,
    });
    const result = await response.json();
    if (result.success) {
      return result.user;
    }
    throw new Error(result.error || 'Failed to get user');
  }

  // Chats
  async getChats(): Promise<Chat[]> {
    const response = await this.request.get(`${BASE_URL}/api/v1/chats`, {
      headers: this.headers,
    });
    const result = await response.json();
    if (result.success) {
      return result.chats || [];
    }
    throw new Error(result.error || 'Failed to get chats');
  }

  async createDirectChat(participantId: string): Promise<Chat> {
    const response = await this.request.post(`${BASE_URL}/api/v1/chats`, {
      headers: this.headers,
      data: {
        type: 'direct',
        participant_id: participantId,
      },
    });
    const result = await response.json();
    if (result.success) {
      return result.chat;
    }
    throw new Error(result.error || 'Failed to create chat');
  }

  async createGroupChat(name: string, participantIds: string[]): Promise<Chat> {
    const response = await this.request.post(`${BASE_URL}/api/v1/chats`, {
      headers: this.headers,
      data: {
        type: 'group',
        name,
        participant_ids: participantIds,
      },
    });
    const result = await response.json();
    if (result.success) {
      return result.chat;
    }
    throw new Error(result.error || 'Failed to create group');
  }

  // Messages
  async getMessages(chatId: string): Promise<Message[]> {
    const response = await this.request.get(`${BASE_URL}/api/v1/chats/${chatId}/messages`, {
      headers: this.headers,
    });
    const result = await response.json();
    if (result.success) {
      return result.messages || [];
    }
    throw new Error(result.error || 'Failed to get messages');
  }

  async sendMessage(chatId: string, content: string): Promise<Message> {
    const response = await this.request.post(`${BASE_URL}/api/v1/chats/${chatId}/messages`, {
      headers: this.headers,
      data: { content },
    });
    const result = await response.json();
    if (result.success) {
      return result.message;
    }
    throw new Error(result.error || 'Failed to send message');
  }

  async markAsRead(chatId: string): Promise<void> {
    await this.request.post(`${BASE_URL}/api/v1/chats/${chatId}/read`, {
      headers: this.headers,
    });
  }

  // Contacts
  async getContacts(): Promise<User[]> {
    const response = await this.request.get(`${BASE_URL}/api/v1/contacts`, {
      headers: this.headers,
    });
    const result = await response.json();
    if (result.success) {
      return result.contacts || [];
    }
    throw new Error(result.error || 'Failed to get contacts');
  }

  async addContact(userId: string): Promise<void> {
    const response = await this.request.post(`${BASE_URL}/api/v1/contacts`, {
      headers: this.headers,
      data: { user_id: userId },
    });
    const result = await response.json();
    if (!result.success) {
      throw new Error(result.error || 'Failed to add contact');
    }
  }

  // Users
  async searchUsers(query: string): Promise<User[]> {
    const response = await this.request.get(`${BASE_URL}/api/v1/users/search?q=${encodeURIComponent(query)}`, {
      headers: this.headers,
    });
    const result = await response.json();
    if (result.success) {
      return result.users || [];
    }
    throw new Error(result.error || 'Failed to search users');
  }

  // Test data setup
  async setupTestData(): Promise<{
    users: Record<string, User>;
    chats: Chat[];
  }> {
    const users: Record<string, User> = {};

    // Create test users
    const testUsers = [
      { username: 'alice', password: 'password123', display_name: 'Alice Smith' },
      { username: 'bob', password: 'password123', display_name: 'Bob Jones' },
      { username: 'charlie', password: 'password123', display_name: 'Charlie Brown' },
    ];

    for (const userData of testUsers) {
      try {
        const { user } = await this.register(userData);
        users[userData.username] = user;
      } catch {
        // User might already exist, try to login
        try {
          const { user } = await this.login(userData.username, userData.password);
          users[userData.username] = user;
        } catch {
          console.warn(`Could not create or login as ${userData.username}`);
        }
      }
    }

    // Create chats between users
    const chats: Chat[] = [];

    if (users.alice && users.bob) {
      await this.login('alice', 'password123');
      try {
        const chat = await this.createDirectChat(users.bob.id);
        chats.push(chat);

        // Send some messages
        await this.sendMessage(chat.id, 'Hey Bob!');
        await this.sendMessage(chat.id, 'How are you?');

        // Login as Bob and reply
        await this.login('bob', 'password123');
        await this.sendMessage(chat.id, 'Hi Alice! I am good.');
        await this.sendMessage(chat.id, 'How about you?');
      } catch (e) {
        console.warn('Could not set up alice-bob chat:', e);
      }
    }

    return { users, chats };
  }
}

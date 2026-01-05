import type { Store } from '../store/types';
import type { User, Session, CreateUser, UpdateUser, LoginInput, UserSettings } from '../models';
import { generateId } from '../utils/id';
import { hashPassword, verifyPassword } from '../utils/password';
import { SESSION_DURATION_MS } from '../utils/cookie';

export class UserService {
  constructor(private store: Store) {}

  async register(input: CreateUser): Promise<{ user: User; session: Session }> {
    // Check if email already exists
    const existing = await this.store.users.getByEmail(input.email);
    if (existing) {
      throw new Error('Email already registered');
    }

    // Hash password
    const passwordHash = await hashPassword(input.password);

    // Create user
    const user = await this.store.users.create({
      id: generateId(),
      email: input.email,
      name: input.name,
      passwordHash,
      settings: defaultUserSettings(),
    });

    // Create session
    const session = await this.createSession(user.id);

    return { user, session };
  }

  async login(input: LoginInput): Promise<{ user: User; session: Session }> {
    // Find user by email
    const user = await this.store.users.getByEmail(input.email);
    if (!user) {
      throw new Error('Invalid email or password');
    }

    // Verify password
    const valid = await verifyPassword(input.password, user.passwordHash);
    if (!valid) {
      throw new Error('Invalid email or password');
    }

    // Create session
    const session = await this.createSession(user.id);

    return { user, session };
  }

  async logout(sessionId: string): Promise<void> {
    await this.store.sessions.deleteById(sessionId);
  }

  async getById(id: string): Promise<User | null> {
    return this.store.users.getById(id);
  }

  async getBySession(sessionId: string): Promise<User | null> {
    const session = await this.store.sessions.getById(sessionId);
    if (!session) return null;

    // Check if session is expired
    if (new Date(session.expiresAt) < new Date()) {
      await this.store.sessions.deleteById(sessionId);
      return null;
    }

    return this.store.users.getById(session.userId);
  }

  async update(id: string, data: UpdateUser): Promise<User> {
    return this.store.users.update(id, data);
  }

  private async createSession(userId: string): Promise<Session> {
    const expiresAt = new Date(Date.now() + SESSION_DURATION_MS).toISOString();
    return this.store.sessions.create({
      id: generateId(),
      userId,
      expiresAt,
    });
  }
}

function defaultUserSettings(): UserSettings {
  return {
    theme: 'system',
    timezone: 'UTC',
    dateFormat: 'MM/DD/YYYY',
    startOfWeek: 0,
    emailDigest: true,
    desktopNotify: true,
  };
}

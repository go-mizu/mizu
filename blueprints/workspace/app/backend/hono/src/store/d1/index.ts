import type {
  Store,
  UserStore,
  SessionStore,
  WorkspaceStore,
  MemberStore,
  PageStore,
  BlockStore,
  DatabaseStore,
  ViewStore,
  CommentStore,
  ShareStore,
  FavoriteStore,
  ActivityStore,
  NotificationStore,
  SyncedBlockStore,
} from '../types';
import type {
  User,
  Session,
  Workspace,
  Member,
  Page,
  Block,
  Database,
  View,
  Comment,
  Share,
  Favorite,
  Activity,
  Notification,
  SyncedBlock,
  PaginatedResult,
  FilterGroup,
  Sort,
} from '../../models';
import { nowISO } from '../../models/common';

// Schema SQL (embedded)
const SCHEMA = `
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  avatar_url TEXT,
  password_hash TEXT NOT NULL,
  settings TEXT DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspaces (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  icon TEXT,
  domain TEXT,
  plan TEXT DEFAULT 'free',
  settings TEXT DEFAULT '{}',
  owner_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS members (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'member',
  created_at TEXT NOT NULL,
  UNIQUE(workspace_id, user_id)
);

CREATE TABLE IF NOT EXISTS pages (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  parent_id TEXT,
  parent_type TEXT NOT NULL DEFAULT 'workspace',
  database_id TEXT,
  row_position REAL,
  title TEXT NOT NULL DEFAULT '',
  icon TEXT,
  cover TEXT,
  cover_y REAL DEFAULT 0.5,
  properties TEXT DEFAULT '{}',
  is_template INTEGER DEFAULT 0,
  is_archived INTEGER DEFAULT 0,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS blocks (
  id TEXT PRIMARY KEY,
  page_id TEXT NOT NULL,
  parent_id TEXT,
  type TEXT NOT NULL,
  content TEXT DEFAULT '{}',
  position REAL NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS databases (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  page_id TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  icon TEXT,
  cover TEXT,
  is_inline INTEGER DEFAULT 0,
  properties TEXT DEFAULT '[]',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS views (
  id TEXT PRIMARY KEY,
  database_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'table',
  filter TEXT,
  sorts TEXT,
  properties TEXT,
  group_by TEXT,
  calendar_by TEXT,
  position REAL NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS comments (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  parent_id TEXT,
  content TEXT NOT NULL,
  author_id TEXT NOT NULL,
  is_resolved INTEGER DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS shares (
  id TEXT PRIMARY KEY,
  page_id TEXT NOT NULL,
  type TEXT NOT NULL,
  permission TEXT NOT NULL DEFAULT 'read',
  user_id TEXT,
  token TEXT,
  password TEXT,
  expires_at TEXT,
  domain TEXT,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS favorites (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  page_id TEXT NOT NULL,
  position REAL NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(user_id, page_id)
);

CREATE TABLE IF NOT EXISTS synced_blocks (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  source_block_id TEXT NOT NULL,
  content TEXT NOT NULL,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS activities (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  metadata TEXT DEFAULT '{}',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT,
  metadata TEXT DEFAULT '{}',
  is_read INTEGER DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_members_workspace_id ON members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_members_user_id ON members(user_id);
CREATE INDEX IF NOT EXISTS idx_pages_workspace_id ON pages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_pages_parent_id ON pages(parent_id);
CREATE INDEX IF NOT EXISTS idx_pages_database_id ON pages(database_id);
CREATE INDEX IF NOT EXISTS idx_blocks_page_id ON blocks(page_id);
CREATE INDEX IF NOT EXISTS idx_blocks_parent_id ON blocks(parent_id);
CREATE INDEX IF NOT EXISTS idx_databases_workspace_id ON databases(workspace_id);
CREATE INDEX IF NOT EXISTS idx_views_database_id ON views(database_id);
CREATE INDEX IF NOT EXISTS idx_comments_target ON comments(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_shares_page_id ON shares(page_id);
CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token);
CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_activities_workspace_id ON activities(workspace_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
`;

// Helper to parse JSON safely
function parseJSON<T>(str: string | null | undefined, defaultValue: T): T {
  if (!str) return defaultValue;
  try {
    return JSON.parse(str) as T;
  } catch {
    return defaultValue;
  }
}

// D1 User Store
class D1UserStore implements UserStore {
  constructor(private db: D1Database) {}

  async create(user: Omit<User, 'createdAt' | 'updatedAt'>): Promise<User> {
    const now = nowISO();
    const result: User = { ...user, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO users (id, email, name, avatar_url, password_hash, settings, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.email,
        result.name,
        result.avatarUrl ?? null,
        result.passwordHash,
        JSON.stringify(result.settings),
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<User | null> {
    const row = await this.db
      .prepare('SELECT * FROM users WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async getByEmail(email: string): Promise<User | null> {
    const row = await this.db
      .prepare('SELECT * FROM users WHERE email = ?')
      .bind(email)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async update(id: string, data: Partial<User>): Promise<User> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.avatarUrl !== undefined) {
      updates.push('avatar_url = ?');
      values.push(data.avatarUrl);
    }
    if (data.settings !== undefined) {
      updates.push('settings = ?');
      values.push(JSON.stringify(data.settings));
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE users SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const user = await this.getById(id);
    if (!user) throw new Error('User not found');
    return user;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM users WHERE id = ?').bind(id).run();
  }

  private mapRow(row: Record<string, unknown>): User {
    return {
      id: row.id as string,
      email: row.email as string,
      name: row.name as string,
      avatarUrl: row.avatar_url as string | undefined,
      passwordHash: row.password_hash as string,
      settings: parseJSON(row.settings as string, {}),
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Session Store
class D1SessionStore implements SessionStore {
  constructor(private db: D1Database) {}

  async create(session: Omit<Session, 'createdAt'>): Promise<Session> {
    const now = nowISO();
    const result: Session = { ...session, createdAt: now };

    await this.db
      .prepare(
        `INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`
      )
      .bind(result.id, result.userId, result.expiresAt, result.createdAt)
      .run();

    return result;
  }

  async getById(id: string): Promise<Session | null> {
    const row = await this.db
      .prepare('SELECT * FROM sessions WHERE id = ?')
      .bind(id)
      .first();

    if (!row) return null;

    return {
      id: row.id as string,
      userId: row.user_id as string,
      expiresAt: row.expires_at as string,
      createdAt: row.created_at as string,
    };
  }

  async deleteById(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM sessions WHERE id = ?').bind(id).run();
  }

  async deleteByUserId(userId: string): Promise<void> {
    await this.db
      .prepare('DELETE FROM sessions WHERE user_id = ?')
      .bind(userId)
      .run();
  }

  async deleteExpired(): Promise<void> {
    await this.db
      .prepare('DELETE FROM sessions WHERE expires_at < ?')
      .bind(nowISO())
      .run();
  }
}

// D1 Workspace Store
class D1WorkspaceStore implements WorkspaceStore {
  constructor(private db: D1Database) {}

  async create(
    workspace: Omit<Workspace, 'createdAt' | 'updatedAt'>
  ): Promise<Workspace> {
    const now = nowISO();
    const result: Workspace = { ...workspace, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO workspaces (id, name, slug, icon, domain, plan, settings, owner_id, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.name,
        result.slug,
        result.icon ?? null,
        result.domain ?? null,
        result.plan,
        JSON.stringify(result.settings),
        result.ownerId,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Workspace | null> {
    const row = await this.db
      .prepare('SELECT * FROM workspaces WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async getBySlug(slug: string): Promise<Workspace | null> {
    const row = await this.db
      .prepare('SELECT * FROM workspaces WHERE slug = ?')
      .bind(slug)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByUser(userId: string): Promise<Workspace[]> {
    const { results } = await this.db
      .prepare(
        `SELECT w.* FROM workspaces w
         INNER JOIN members m ON m.workspace_id = w.id
         WHERE m.user_id = ?
         ORDER BY w.created_at DESC`
      )
      .bind(userId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Workspace>): Promise<Workspace> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.slug !== undefined) {
      updates.push('slug = ?');
      values.push(data.slug);
    }
    if (data.icon !== undefined) {
      updates.push('icon = ?');
      values.push(data.icon);
    }
    if (data.domain !== undefined) {
      updates.push('domain = ?');
      values.push(data.domain);
    }
    if (data.settings !== undefined) {
      updates.push('settings = ?');
      values.push(JSON.stringify(data.settings));
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE workspaces SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const workspace = await this.getById(id);
    if (!workspace) throw new Error('Workspace not found');
    return workspace;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM workspaces WHERE id = ?').bind(id).run();
  }

  private mapRow(row: Record<string, unknown>): Workspace {
    return {
      id: row.id as string,
      name: row.name as string,
      slug: row.slug as string,
      icon: row.icon as string | undefined,
      domain: row.domain as string | undefined,
      plan: row.plan as Workspace['plan'],
      settings: parseJSON(row.settings as string, {}),
      ownerId: row.owner_id as string,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Member Store
class D1MemberStore implements MemberStore {
  constructor(private db: D1Database) {}

  async create(member: Omit<Member, 'createdAt'>): Promise<Member> {
    const now = nowISO();
    const result: Member = { ...member, createdAt: now };

    await this.db
      .prepare(
        `INSERT INTO members (id, workspace_id, user_id, role, created_at) VALUES (?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.workspaceId,
        result.userId,
        result.role,
        result.createdAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Member | null> {
    const row = await this.db
      .prepare('SELECT * FROM members WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async getByWorkspaceAndUser(
    workspaceId: string,
    userId: string
  ): Promise<Member | null> {
    const row = await this.db
      .prepare('SELECT * FROM members WHERE workspace_id = ? AND user_id = ?')
      .bind(workspaceId, userId)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(workspaceId: string): Promise<Member[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM members WHERE workspace_id = ?')
      .bind(workspaceId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async listByUser(userId: string): Promise<Member[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM members WHERE user_id = ?')
      .bind(userId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Member>): Promise<Member> {
    if (data.role !== undefined) {
      await this.db
        .prepare('UPDATE members SET role = ? WHERE id = ?')
        .bind(data.role, id)
        .run();
    }

    const member = await this.getById(id);
    if (!member) throw new Error('Member not found');
    return member;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM members WHERE id = ?').bind(id).run();
  }

  private mapRow(row: Record<string, unknown>): Member {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      userId: row.user_id as string,
      role: row.role as Member['role'],
      createdAt: row.created_at as string,
    };
  }
}

// D1 Page Store
class D1PageStore implements PageStore {
  constructor(private db: D1Database) {}

  async create(page: Omit<Page, 'createdAt' | 'updatedAt'>): Promise<Page> {
    const now = nowISO();
    const result: Page = { ...page, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO pages (id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, properties, is_template, is_archived, created_by, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.workspaceId,
        result.parentId ?? null,
        result.parentType,
        result.databaseId ?? null,
        result.rowPosition ?? null,
        result.title,
        result.icon ?? null,
        result.cover ?? null,
        result.coverY ?? 0.5,
        JSON.stringify(result.properties),
        result.isTemplate ? 1 : 0,
        result.isArchived ? 1 : 0,
        result.createdBy,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Page | null> {
    const row = await this.db
      .prepare('SELECT * FROM pages WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(
    workspaceId: string,
    options?: { parentId?: string | null; includeArchived?: boolean }
  ): Promise<Page[]> {
    let sql = 'SELECT * FROM pages WHERE workspace_id = ? AND database_id IS NULL';
    const params: unknown[] = [workspaceId];

    if (options?.parentId !== undefined) {
      if (options.parentId === null) {
        sql += ' AND parent_id IS NULL';
      } else {
        sql += ' AND parent_id = ?';
        params.push(options.parentId);
      }
    }

    if (!options?.includeArchived) {
      sql += ' AND is_archived = 0';
    }

    sql += ' ORDER BY created_at DESC';

    const { results } = await this.db
      .prepare(sql)
      .bind(...params)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async listByParent(parentId: string, parentType: string): Promise<Page[]> {
    const { results } = await this.db
      .prepare(
        'SELECT * FROM pages WHERE parent_id = ? AND parent_type = ? AND is_archived = 0 ORDER BY created_at DESC'
      )
      .bind(parentId, parentType)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async listByDatabase(
    databaseId: string,
    options?: {
      cursor?: string;
      limit?: number;
      filter?: FilterGroup;
      sorts?: Sort[];
    }
  ): Promise<PaginatedResult<Page>> {
    const limit = options?.limit ?? 50;
    let sql = 'SELECT * FROM pages WHERE database_id = ? AND is_archived = 0';
    const params: unknown[] = [databaseId];

    if (options?.cursor) {
      sql += ' AND row_position > ?';
      params.push(parseFloat(options.cursor));
    }

    // TODO: Apply filters and sorts
    sql += ' ORDER BY row_position ASC LIMIT ?';
    params.push(limit + 1);

    const { results } = await this.db
      .prepare(sql)
      .bind(...params)
      .all();

    const rows = (results ?? []).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
    const items = hasMore ? rows.slice(0, limit) : rows;
    const nextCursor = hasMore
      ? String(items[items.length - 1]?.rowPosition ?? 0)
      : undefined;

    return { items, nextCursor, hasMore };
  }

  async update(id: string, data: Partial<Page>): Promise<Page> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.title !== undefined) {
      updates.push('title = ?');
      values.push(data.title);
    }
    if (data.icon !== undefined) {
      updates.push('icon = ?');
      values.push(data.icon);
    }
    if (data.cover !== undefined) {
      updates.push('cover = ?');
      values.push(data.cover);
    }
    if (data.coverY !== undefined) {
      updates.push('cover_y = ?');
      values.push(data.coverY);
    }
    if (data.properties !== undefined) {
      updates.push('properties = ?');
      values.push(JSON.stringify(data.properties));
    }
    if (data.isArchived !== undefined) {
      updates.push('is_archived = ?');
      values.push(data.isArchived ? 1 : 0);
    }
    if (data.parentId !== undefined) {
      updates.push('parent_id = ?');
      values.push(data.parentId);
    }
    if (data.parentType !== undefined) {
      updates.push('parent_type = ?');
      values.push(data.parentType);
    }
    if (data.rowPosition !== undefined) {
      updates.push('row_position = ?');
      values.push(data.rowPosition);
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE pages SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const page = await this.getById(id);
    if (!page) throw new Error('Page not found');
    return page;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM pages WHERE id = ?').bind(id).run();
  }

  async getHierarchy(
    id: string
  ): Promise<{ id: string; title: string; icon?: string | null }[]> {
    const breadcrumb: { id: string; title: string; icon?: string | null }[] = [];
    let currentId: string | null = id;

    while (currentId) {
      const page = await this.getById(currentId);
      if (!page) break;

      breadcrumb.unshift({ id: page.id, title: page.title, icon: page.icon });
      currentId = page.parentId ?? null;
    }

    return breadcrumb;
  }

  private mapRow(row: Record<string, unknown>): Page {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      parentId: row.parent_id as string | undefined,
      parentType: row.parent_type as Page['parentType'],
      databaseId: row.database_id as string | undefined,
      rowPosition: row.row_position as number | undefined,
      title: row.title as string,
      icon: row.icon as string | undefined,
      cover: row.cover as string | undefined,
      coverY: row.cover_y as number,
      properties: parseJSON(row.properties as string, {}),
      isTemplate: Boolean(row.is_template),
      isArchived: Boolean(row.is_archived),
      createdBy: row.created_by as string,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Block Store
class D1BlockStore implements BlockStore {
  constructor(private db: D1Database) {}

  async create(block: Omit<Block, 'createdAt' | 'updatedAt'>): Promise<Block> {
    const now = nowISO();
    const result: Block = { ...block, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO blocks (id, page_id, parent_id, type, content, position, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.pageId,
        result.parentId ?? null,
        result.type,
        JSON.stringify(result.content),
        result.position,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Block | null> {
    const row = await this.db
      .prepare('SELECT * FROM blocks WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByPage(pageId: string): Promise<Block[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM blocks WHERE page_id = ? ORDER BY position ASC')
      .bind(pageId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async listByParent(parentId: string): Promise<Block[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM blocks WHERE parent_id = ? ORDER BY position ASC')
      .bind(parentId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Block>): Promise<Block> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.type !== undefined) {
      updates.push('type = ?');
      values.push(data.type);
    }
    if (data.content !== undefined) {
      updates.push('content = ?');
      values.push(JSON.stringify(data.content));
    }
    if (data.position !== undefined) {
      updates.push('position = ?');
      values.push(data.position);
    }
    if (data.parentId !== undefined) {
      updates.push('parent_id = ?');
      values.push(data.parentId);
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE blocks SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const block = await this.getById(id);
    if (!block) throw new Error('Block not found');
    return block;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM blocks WHERE id = ?').bind(id).run();
  }

  async deleteByPage(pageId: string): Promise<void> {
    await this.db
      .prepare('DELETE FROM blocks WHERE page_id = ?')
      .bind(pageId)
      .run();
  }

  async getMaxPosition(pageId: string, parentId?: string): Promise<number> {
    const sql = parentId
      ? 'SELECT MAX(position) as max FROM blocks WHERE page_id = ? AND parent_id = ?'
      : 'SELECT MAX(position) as max FROM blocks WHERE page_id = ? AND parent_id IS NULL';

    const row = parentId
      ? await this.db.prepare(sql).bind(pageId, parentId).first()
      : await this.db.prepare(sql).bind(pageId).first();

    return ((row?.max as number) ?? 0) + 1;
  }

  async batchUpsert(pageId: string, blocks: Block[]): Promise<void> {
    // Delete existing blocks
    await this.deleteByPage(pageId);

    // Insert new blocks
    for (const block of blocks) {
      await this.db
        .prepare(
          `INSERT INTO blocks (id, page_id, parent_id, type, content, position, created_at, updated_at)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
        )
        .bind(
          block.id,
          block.pageId,
          block.parentId ?? null,
          block.type,
          JSON.stringify(block.content),
          block.position,
          block.createdAt,
          block.updatedAt
        )
        .run();
    }
  }

  private mapRow(row: Record<string, unknown>): Block {
    return {
      id: row.id as string,
      pageId: row.page_id as string,
      parentId: row.parent_id as string | undefined,
      type: row.type as Block['type'],
      content: parseJSON(row.content as string, {}),
      position: row.position as number,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Database Store
class D1DatabaseStore implements DatabaseStore {
  constructor(private db: D1Database) {}

  async create(
    database: Omit<Database, 'createdAt' | 'updatedAt'>
  ): Promise<Database> {
    const now = nowISO();
    const result: Database = { ...database, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO databases (id, workspace_id, page_id, title, icon, cover, is_inline, properties, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.workspaceId,
        result.pageId,
        result.title,
        result.icon ?? null,
        result.cover ?? null,
        result.isInline ? 1 : 0,
        JSON.stringify(result.properties),
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Database | null> {
    const row = await this.db
      .prepare('SELECT * FROM databases WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async getByPageId(pageId: string): Promise<Database | null> {
    const row = await this.db
      .prepare('SELECT * FROM databases WHERE page_id = ?')
      .bind(pageId)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(workspaceId: string): Promise<Database[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM databases WHERE workspace_id = ?')
      .bind(workspaceId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Database>): Promise<Database> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.title !== undefined) {
      updates.push('title = ?');
      values.push(data.title);
    }
    if (data.icon !== undefined) {
      updates.push('icon = ?');
      values.push(data.icon);
    }
    if (data.cover !== undefined) {
      updates.push('cover = ?');
      values.push(data.cover);
    }
    if (data.properties !== undefined) {
      updates.push('properties = ?');
      values.push(JSON.stringify(data.properties));
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE databases SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const database = await this.getById(id);
    if (!database) throw new Error('Database not found');
    return database;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM databases WHERE id = ?').bind(id).run();
  }

  private mapRow(row: Record<string, unknown>): Database {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      pageId: row.page_id as string,
      title: row.title as string,
      icon: row.icon as string | undefined,
      cover: row.cover as string | undefined,
      isInline: Boolean(row.is_inline),
      properties: parseJSON(row.properties as string, []),
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 View Store
class D1ViewStore implements ViewStore {
  constructor(private db: D1Database) {}

  async create(view: Omit<View, 'createdAt' | 'updatedAt'>): Promise<View> {
    const now = nowISO();
    const result: View = { ...view, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO views (id, database_id, name, type, filter, sorts, properties, group_by, calendar_by, position, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.databaseId,
        result.name,
        result.type,
        result.filter ? JSON.stringify(result.filter) : null,
        result.sorts ? JSON.stringify(result.sorts) : null,
        result.properties ? JSON.stringify(result.properties) : null,
        result.groupBy ?? null,
        result.calendarBy ?? null,
        result.position,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<View | null> {
    const row = await this.db
      .prepare('SELECT * FROM views WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByDatabase(databaseId: string): Promise<View[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM views WHERE database_id = ? ORDER BY position ASC')
      .bind(databaseId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<View>): Promise<View> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.type !== undefined) {
      updates.push('type = ?');
      values.push(data.type);
    }
    if (data.filter !== undefined) {
      updates.push('filter = ?');
      values.push(data.filter ? JSON.stringify(data.filter) : null);
    }
    if (data.sorts !== undefined) {
      updates.push('sorts = ?');
      values.push(data.sorts ? JSON.stringify(data.sorts) : null);
    }
    if (data.properties !== undefined) {
      updates.push('properties = ?');
      values.push(data.properties ? JSON.stringify(data.properties) : null);
    }
    if (data.groupBy !== undefined) {
      updates.push('group_by = ?');
      values.push(data.groupBy);
    }
    if (data.calendarBy !== undefined) {
      updates.push('calendar_by = ?');
      values.push(data.calendarBy);
    }
    if (data.position !== undefined) {
      updates.push('position = ?');
      values.push(data.position);
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE views SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const view = await this.getById(id);
    if (!view) throw new Error('View not found');
    return view;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM views WHERE id = ?').bind(id).run();
  }

  async getMaxPosition(databaseId: string): Promise<number> {
    const row = await this.db
      .prepare('SELECT MAX(position) as max FROM views WHERE database_id = ?')
      .bind(databaseId)
      .first();

    return ((row?.max as number) ?? 0) + 1;
  }

  private mapRow(row: Record<string, unknown>): View {
    return {
      id: row.id as string,
      databaseId: row.database_id as string,
      name: row.name as string,
      type: row.type as View['type'],
      filter: parseJSON(row.filter as string, null),
      sorts: parseJSON(row.sorts as string, null),
      properties: parseJSON(row.properties as string, null),
      groupBy: row.group_by as string | undefined,
      calendarBy: row.calendar_by as string | undefined,
      position: row.position as number,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Comment Store
class D1CommentStore implements CommentStore {
  constructor(private db: D1Database) {}

  async create(
    comment: Omit<Comment, 'createdAt' | 'updatedAt'>
  ): Promise<Comment> {
    const now = nowISO();
    const result: Comment = { ...comment, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO comments (id, workspace_id, target_type, target_id, parent_id, content, author_id, is_resolved, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.workspaceId,
        result.targetType,
        result.targetId,
        result.parentId ?? null,
        JSON.stringify(result.content),
        result.authorId,
        result.isResolved ? 1 : 0,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Comment | null> {
    const row = await this.db
      .prepare('SELECT * FROM comments WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByTarget(targetType: string, targetId: string): Promise<Comment[]> {
    const { results } = await this.db
      .prepare(
        'SELECT * FROM comments WHERE target_type = ? AND target_id = ? ORDER BY created_at ASC'
      )
      .bind(targetType, targetId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async listByPage(pageId: string): Promise<Comment[]> {
    const { results } = await this.db
      .prepare(
        `SELECT * FROM comments WHERE target_type = 'page' AND target_id = ? ORDER BY created_at ASC`
      )
      .bind(pageId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Comment>): Promise<Comment> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.content !== undefined) {
      updates.push('content = ?');
      values.push(JSON.stringify(data.content));
    }
    if (data.isResolved !== undefined) {
      updates.push('is_resolved = ?');
      values.push(data.isResolved ? 1 : 0);
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE comments SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const comment = await this.getById(id);
    if (!comment) throw new Error('Comment not found');
    return comment;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM comments WHERE id = ?').bind(id).run();
  }

  private mapRow(row: Record<string, unknown>): Comment {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      targetType: row.target_type as Comment['targetType'],
      targetId: row.target_id as string,
      parentId: row.parent_id as string | undefined,
      content: parseJSON(row.content as string, []),
      authorId: row.author_id as string,
      isResolved: Boolean(row.is_resolved),
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Share Store
class D1ShareStore implements ShareStore {
  constructor(private db: D1Database) {}

  async create(share: Omit<Share, 'createdAt' | 'updatedAt'>): Promise<Share> {
    const now = nowISO();
    const result: Share = { ...share, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO shares (id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.pageId,
        result.type,
        result.permission,
        result.userId ?? null,
        result.token ?? null,
        result.password ?? null,
        result.expiresAt ?? null,
        result.domain ?? null,
        result.createdBy,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Share | null> {
    const row = await this.db
      .prepare('SELECT * FROM shares WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async getByToken(token: string): Promise<Share | null> {
    const row = await this.db
      .prepare('SELECT * FROM shares WHERE token = ?')
      .bind(token)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByPage(pageId: string): Promise<Share[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM shares WHERE page_id = ?')
      .bind(pageId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Share>): Promise<Share> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.permission !== undefined) {
      updates.push('permission = ?');
      values.push(data.permission);
    }
    if (data.password !== undefined) {
      updates.push('password = ?');
      values.push(data.password);
    }
    if (data.expiresAt !== undefined) {
      updates.push('expires_at = ?');
      values.push(data.expiresAt);
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE shares SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const share = await this.getById(id);
    if (!share) throw new Error('Share not found');
    return share;
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM shares WHERE id = ?').bind(id).run();
  }

  private mapRow(row: Record<string, unknown>): Share {
    return {
      id: row.id as string,
      pageId: row.page_id as string,
      type: row.type as Share['type'],
      permission: row.permission as Share['permission'],
      userId: row.user_id as string | undefined,
      token: row.token as string | undefined,
      password: row.password as string | undefined,
      expiresAt: row.expires_at as string | undefined,
      domain: row.domain as string | undefined,
      createdBy: row.created_by as string,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// D1 Favorite Store
class D1FavoriteStore implements FavoriteStore {
  constructor(private db: D1Database) {}

  async create(favorite: Omit<Favorite, 'createdAt'>): Promise<Favorite> {
    const now = nowISO();
    const result: Favorite = { ...favorite, createdAt: now };

    await this.db
      .prepare(
        `INSERT INTO favorites (id, user_id, page_id, position, created_at) VALUES (?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.userId,
        result.pageId,
        result.position,
        result.createdAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Favorite | null> {
    const row = await this.db
      .prepare('SELECT * FROM favorites WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async getByUserAndPage(
    userId: string,
    pageId: string
  ): Promise<Favorite | null> {
    const row = await this.db
      .prepare('SELECT * FROM favorites WHERE user_id = ? AND page_id = ?')
      .bind(userId, pageId)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByUser(userId: string, workspaceId: string): Promise<Favorite[]> {
    const { results } = await this.db
      .prepare(
        `SELECT f.* FROM favorites f
         INNER JOIN pages p ON p.id = f.page_id
         WHERE f.user_id = ? AND p.workspace_id = ?
         ORDER BY f.position ASC`
      )
      .bind(userId, workspaceId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare('DELETE FROM favorites WHERE id = ?').bind(id).run();
  }

  async deleteByUserAndPage(userId: string, pageId: string): Promise<void> {
    await this.db
      .prepare('DELETE FROM favorites WHERE user_id = ? AND page_id = ?')
      .bind(userId, pageId)
      .run();
  }

  async getMaxPosition(userId: string): Promise<number> {
    const row = await this.db
      .prepare('SELECT MAX(position) as max FROM favorites WHERE user_id = ?')
      .bind(userId)
      .first();

    return ((row?.max as number) ?? 0) + 1;
  }

  private mapRow(row: Record<string, unknown>): Favorite {
    return {
      id: row.id as string,
      userId: row.user_id as string,
      pageId: row.page_id as string,
      position: row.position as number,
      createdAt: row.created_at as string,
    };
  }
}

// D1 Activity Store
class D1ActivityStore implements ActivityStore {
  constructor(private db: D1Database) {}

  async create(activity: Omit<Activity, 'createdAt'>): Promise<Activity> {
    const now = nowISO();
    const result: Activity = { ...activity, createdAt: now };

    await this.db
      .prepare(
        `INSERT INTO activities (id, workspace_id, user_id, action, target_type, target_id, metadata, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.workspaceId,
        result.userId,
        result.action,
        result.targetType,
        result.targetId,
        JSON.stringify(result.metadata),
        result.createdAt
      )
      .run();

    return result;
  }

  async listByWorkspace(
    workspaceId: string,
    options?: { cursor?: string; limit?: number }
  ): Promise<PaginatedResult<Activity>> {
    const limit = options?.limit ?? 50;
    let sql = 'SELECT * FROM activities WHERE workspace_id = ?';
    const params: unknown[] = [workspaceId];

    if (options?.cursor) {
      sql += ' AND created_at < ?';
      params.push(options.cursor);
    }

    sql += ' ORDER BY created_at DESC LIMIT ?';
    params.push(limit + 1);

    const { results } = await this.db
      .prepare(sql)
      .bind(...params)
      .all();

    const rows = (results ?? []).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
    const items = hasMore ? rows.slice(0, limit) : rows;
    const nextCursor = hasMore ? items[items.length - 1]?.createdAt : undefined;

    return { items, nextCursor, hasMore };
  }

  async listByUser(
    userId: string,
    options?: { cursor?: string; limit?: number }
  ): Promise<PaginatedResult<Activity>> {
    const limit = options?.limit ?? 50;
    let sql = 'SELECT * FROM activities WHERE user_id = ?';
    const params: unknown[] = [userId];

    if (options?.cursor) {
      sql += ' AND created_at < ?';
      params.push(options.cursor);
    }

    sql += ' ORDER BY created_at DESC LIMIT ?';
    params.push(limit + 1);

    const { results } = await this.db
      .prepare(sql)
      .bind(...params)
      .all();

    const rows = (results ?? []).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
    const items = hasMore ? rows.slice(0, limit) : rows;
    const nextCursor = hasMore ? items[items.length - 1]?.createdAt : undefined;

    return { items, nextCursor, hasMore };
  }

  private mapRow(row: Record<string, unknown>): Activity {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      userId: row.user_id as string,
      action: row.action as string,
      targetType: row.target_type as string,
      targetId: row.target_id as string,
      metadata: parseJSON(row.metadata as string, {}),
      createdAt: row.created_at as string,
    };
  }
}

// D1 Notification Store
class D1NotificationStore implements NotificationStore {
  constructor(private db: D1Database) {}

  async create(
    notification: Omit<Notification, 'createdAt'>
  ): Promise<Notification> {
    const now = nowISO();
    const result: Notification = { ...notification, createdAt: now };

    await this.db
      .prepare(
        `INSERT INTO notifications (id, user_id, type, title, message, metadata, is_read, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.userId,
        result.type,
        result.title,
        result.message ?? null,
        JSON.stringify(result.metadata),
        result.isRead ? 1 : 0,
        result.createdAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<Notification | null> {
    const row = await this.db
      .prepare('SELECT * FROM notifications WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByUser(
    userId: string,
    options?: { unreadOnly?: boolean; cursor?: string; limit?: number }
  ): Promise<PaginatedResult<Notification>> {
    const limit = options?.limit ?? 50;
    let sql = 'SELECT * FROM notifications WHERE user_id = ?';
    const params: unknown[] = [userId];

    if (options?.unreadOnly) {
      sql += ' AND is_read = 0';
    }

    if (options?.cursor) {
      sql += ' AND created_at < ?';
      params.push(options.cursor);
    }

    sql += ' ORDER BY created_at DESC LIMIT ?';
    params.push(limit + 1);

    const { results } = await this.db
      .prepare(sql)
      .bind(...params)
      .all();

    const rows = (results ?? []).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
    const items = hasMore ? rows.slice(0, limit) : rows;
    const nextCursor = hasMore ? items[items.length - 1]?.createdAt : undefined;

    return { items, nextCursor, hasMore };
  }

  async markAsRead(id: string): Promise<void> {
    await this.db
      .prepare('UPDATE notifications SET is_read = 1 WHERE id = ?')
      .bind(id)
      .run();
  }

  async markAllAsRead(userId: string): Promise<void> {
    await this.db
      .prepare('UPDATE notifications SET is_read = 1 WHERE user_id = ?')
      .bind(userId)
      .run();
  }

  async delete(id: string): Promise<void> {
    await this.db
      .prepare('DELETE FROM notifications WHERE id = ?')
      .bind(id)
      .run();
  }

  private mapRow(row: Record<string, unknown>): Notification {
    return {
      id: row.id as string,
      userId: row.user_id as string,
      type: row.type as string,
      title: row.title as string,
      message: row.message as string | undefined,
      metadata: parseJSON(row.metadata as string, {}),
      isRead: Boolean(row.is_read),
      createdAt: row.created_at as string,
    };
  }
}

// D1 Synced Block Store
class D1SyncedBlockStore implements SyncedBlockStore {
  constructor(private db: D1Database) {}

  async create(
    syncedBlock: Omit<SyncedBlock, 'createdAt' | 'updatedAt'>
  ): Promise<SyncedBlock> {
    const now = nowISO();
    const result: SyncedBlock = { ...syncedBlock, createdAt: now, updatedAt: now };

    await this.db
      .prepare(
        `INSERT INTO synced_blocks (id, workspace_id, source_block_id, content, created_by, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        result.id,
        result.workspaceId,
        result.sourceBlockId,
        JSON.stringify(result.content),
        result.createdBy,
        result.createdAt,
        result.updatedAt
      )
      .run();

    return result;
  }

  async getById(id: string): Promise<SyncedBlock | null> {
    const row = await this.db
      .prepare('SELECT * FROM synced_blocks WHERE id = ?')
      .bind(id)
      .first();

    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(workspaceId: string): Promise<SyncedBlock[]> {
    const { results } = await this.db
      .prepare('SELECT * FROM synced_blocks WHERE workspace_id = ?')
      .bind(workspaceId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async listByPage(pageId: string): Promise<SyncedBlock[]> {
    // Get synced blocks that are referenced in blocks of this page
    const { results } = await this.db
      .prepare(
        `SELECT sb.* FROM synced_blocks sb
         INNER JOIN blocks b ON json_extract(b.content, '$.syncedBlockId') = sb.id
         WHERE b.page_id = ?`
      )
      .bind(pageId)
      .all();

    return (results ?? []).map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<SyncedBlock>): Promise<SyncedBlock> {
    const now = nowISO();
    const updates: string[] = ['updated_at = ?'];
    const values: unknown[] = [now];

    if (data.content !== undefined) {
      updates.push('content = ?');
      values.push(JSON.stringify(data.content));
    }

    values.push(id);

    await this.db
      .prepare(`UPDATE synced_blocks SET ${updates.join(', ')} WHERE id = ?`)
      .bind(...values)
      .run();

    const syncedBlock = await this.getById(id);
    if (!syncedBlock) throw new Error('Synced block not found');
    return syncedBlock;
  }

  async delete(id: string): Promise<void> {
    await this.db
      .prepare('DELETE FROM synced_blocks WHERE id = ?')
      .bind(id)
      .run();
  }

  private mapRow(row: Record<string, unknown>): SyncedBlock {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      sourceBlockId: row.source_block_id as string,
      content: parseJSON(row.content as string, null),
      createdBy: row.created_by as string,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    };
  }
}

// Main D1 Store
export class D1Store implements Store {
  users: UserStore;
  sessions: SessionStore;
  workspaces: WorkspaceStore;
  members: MemberStore;
  pages: PageStore;
  blocks: BlockStore;
  databases: DatabaseStore;
  views: ViewStore;
  comments: CommentStore;
  shares: ShareStore;
  favorites: FavoriteStore;
  activities: ActivityStore;
  notifications: NotificationStore;
  syncedBlocks: SyncedBlockStore;

  constructor(private db: D1Database) {
    this.users = new D1UserStore(db);
    this.sessions = new D1SessionStore(db);
    this.workspaces = new D1WorkspaceStore(db);
    this.members = new D1MemberStore(db);
    this.pages = new D1PageStore(db);
    this.blocks = new D1BlockStore(db);
    this.databases = new D1DatabaseStore(db);
    this.views = new D1ViewStore(db);
    this.comments = new D1CommentStore(db);
    this.shares = new D1ShareStore(db);
    this.favorites = new D1FavoriteStore(db);
    this.activities = new D1ActivityStore(db);
    this.notifications = new D1NotificationStore(db);
    this.syncedBlocks = new D1SyncedBlockStore(db);
  }

  async init(): Promise<void> {
    // Execute schema SQL statements one by one
    const statements = SCHEMA.split(';')
      .map((s) => s.trim())
      .filter((s) => s.length > 0);

    for (const statement of statements) {
      await this.db.prepare(statement).run();
    }
  }

  async close(): Promise<void> {
    // D1 doesn't require explicit close
  }
}

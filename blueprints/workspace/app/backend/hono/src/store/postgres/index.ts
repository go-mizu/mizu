// PostgreSQL Store Implementation using postgres.js
// High-performance, fully async PostgreSQL driver

import type postgres from 'postgres';
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
  Database as DbModel,
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

// Dynamically import postgres
let Postgres: typeof import('postgres').default;

// Helper to get current ISO timestamp
function nowISO(): string {
  return new Date().toISOString();
}

// Helper to convert PostgreSQL timestamp to ISO string
function toISO(date: Date | string | null | undefined): string {
  if (!date) return nowISO();
  if (date instanceof Date) return date.toISOString();
  return date;
}

// PostgreSQL User Store
class PostgresUserStore implements UserStore {
  constructor(private sql: postgres.Sql) {}

  async create(user: Omit<User, 'createdAt' | 'updatedAt'>): Promise<User> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO users (id, email, name, avatar_url, password_hash, settings, created_at, updated_at)
      VALUES (${user.id}, ${user.email}, ${user.name}, ${user.avatarUrl ?? null},
              ${user.passwordHash}, ${JSON.stringify(user.settings)}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<User | null> {
    const [row] = await this.sql`SELECT * FROM users WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async getByEmail(email: string): Promise<User | null> {
    const [row] = await this.sql`SELECT * FROM users WHERE email = ${email}`;
    return row ? this.mapRow(row) : null;
  }

  async update(id: string, data: Partial<User>): Promise<User> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name');
      values.push(data.name);
    }
    if (data.avatarUrl !== undefined) {
      updates.push('avatar_url');
      values.push(data.avatarUrl);
    }
    if (data.settings !== undefined) {
      updates.push('settings');
      values.push(JSON.stringify(data.settings));
    }

    // Build dynamic update
    if (updates.length === 0) {
      const user = await this.getById(id);
      if (!user) throw new Error('User not found');
      return user;
    }

    const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
    await this.sql.unsafe(
      `UPDATE users SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
      [now, ...values, id]
    );

    const user = await this.getById(id);
    if (!user) throw new Error('User not found');
    return user;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM users WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): User {
    return {
      id: row.id as string,
      email: row.email as string,
      name: row.name as string,
      avatarUrl: row.avatar_url as string | undefined,
      passwordHash: row.password_hash as string,
      settings: typeof row.settings === 'string' ? JSON.parse(row.settings) : (row.settings ?? {}),
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Session Store
class PostgresSessionStore implements SessionStore {
  constructor(private sql: postgres.Sql) {}

  async create(session: Omit<Session, 'createdAt'>): Promise<Session> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO sessions (id, user_id, expires_at, created_at)
      VALUES (${session.id}, ${session.userId}, ${session.expiresAt}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Session | null> {
    const [row] = await this.sql`SELECT * FROM sessions WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async deleteById(id: string): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE id = ${id}`;
  }

  async deleteByUserId(userId: string): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE user_id = ${userId}`;
  }

  async deleteExpired(): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE expires_at < ${nowISO()}`;
  }

  private mapRow(row: Record<string, unknown>): Session {
    return {
      id: row.id as string,
      userId: row.user_id as string,
      expiresAt: toISO(row.expires_at as Date),
      createdAt: toISO(row.created_at as Date),
    };
  }
}

// PostgreSQL Workspace Store
class PostgresWorkspaceStore implements WorkspaceStore {
  constructor(private sql: postgres.Sql) {}

  async create(workspace: Omit<Workspace, 'createdAt' | 'updatedAt'>): Promise<Workspace> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO workspaces (id, name, slug, icon, domain, plan, settings, owner_id, created_at, updated_at)
      VALUES (${workspace.id}, ${workspace.name}, ${workspace.slug}, ${workspace.icon ?? null},
              ${workspace.domain ?? null}, ${workspace.plan}, ${JSON.stringify(workspace.settings)},
              ${workspace.ownerId}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Workspace | null> {
    const [row] = await this.sql`SELECT * FROM workspaces WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async getBySlug(slug: string): Promise<Workspace | null> {
    const [row] = await this.sql`SELECT * FROM workspaces WHERE slug = ${slug}`;
    return row ? this.mapRow(row) : null;
  }

  async listByUser(userId: string): Promise<Workspace[]> {
    const rows = await this.sql`
      SELECT w.* FROM workspaces w
      INNER JOIN members m ON m.workspace_id = w.id
      WHERE m.user_id = ${userId}
      ORDER BY w.created_at DESC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Workspace>): Promise<Workspace> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name');
      values.push(data.name);
    }
    if (data.slug !== undefined) {
      updates.push('slug');
      values.push(data.slug);
    }
    if (data.icon !== undefined) {
      updates.push('icon');
      values.push(data.icon);
    }
    if (data.domain !== undefined) {
      updates.push('domain');
      values.push(data.domain);
    }
    if (data.settings !== undefined) {
      updates.push('settings');
      values.push(JSON.stringify(data.settings));
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE workspaces SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const workspace = await this.getById(id);
    if (!workspace) throw new Error('Workspace not found');
    return workspace;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM workspaces WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): Workspace {
    return {
      id: row.id as string,
      name: row.name as string,
      slug: row.slug as string,
      icon: row.icon as string | undefined,
      domain: row.domain as string | undefined,
      plan: row.plan as Workspace['plan'],
      settings: typeof row.settings === 'string' ? JSON.parse(row.settings) : (row.settings ?? {}),
      ownerId: row.owner_id as string,
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Member Store
class PostgresMemberStore implements MemberStore {
  constructor(private sql: postgres.Sql) {}

  async create(member: Omit<Member, 'createdAt'>): Promise<Member> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO members (id, workspace_id, user_id, role, created_at)
      VALUES (${member.id}, ${member.workspaceId}, ${member.userId}, ${member.role}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Member | null> {
    const [row] = await this.sql`SELECT * FROM members WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async getByWorkspaceAndUser(workspaceId: string, userId: string): Promise<Member | null> {
    const [row] = await this.sql`
      SELECT * FROM members WHERE workspace_id = ${workspaceId} AND user_id = ${userId}
    `;
    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(workspaceId: string): Promise<Member[]> {
    const rows = await this.sql`SELECT * FROM members WHERE workspace_id = ${workspaceId}`;
    return rows.map((row) => this.mapRow(row));
  }

  async listByUser(userId: string): Promise<Member[]> {
    const rows = await this.sql`SELECT * FROM members WHERE user_id = ${userId}`;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Member>): Promise<Member> {
    if (data.role !== undefined) {
      await this.sql`UPDATE members SET role = ${data.role} WHERE id = ${id}`;
    }

    const member = await this.getById(id);
    if (!member) throw new Error('Member not found');
    return member;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM members WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): Member {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      userId: row.user_id as string,
      role: row.role as Member['role'],
      createdAt: toISO(row.created_at as Date),
    };
  }
}

// PostgreSQL Page Store
class PostgresPageStore implements PageStore {
  constructor(private sql: postgres.Sql) {}

  async create(page: Omit<Page, 'createdAt' | 'updatedAt'>): Promise<Page> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO pages (id, workspace_id, parent_id, parent_type, database_id, row_position,
        title, icon, cover, cover_y, properties, is_template, is_archived, created_by, created_at, updated_at)
      VALUES (${page.id}, ${page.workspaceId}, ${page.parentId ?? null}, ${page.parentType},
        ${page.databaseId ?? null}, ${page.rowPosition ?? null}, ${page.title}, ${page.icon ?? null},
        ${page.cover ?? null}, ${page.coverY ?? 0.5}, ${JSON.stringify(page.properties)},
        ${page.isTemplate}, ${page.isArchived}, ${page.createdBy}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Page | null> {
    const [row] = await this.sql`SELECT * FROM pages WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(
    workspaceId: string,
    options?: { parentId?: string | null; includeArchived?: boolean }
  ): Promise<Page[]> {
    let rows;
    if (options?.parentId !== undefined) {
      if (options.parentId === null) {
        if (options?.includeArchived) {
          rows = await this.sql`
            SELECT * FROM pages
            WHERE workspace_id = ${workspaceId} AND database_id IS NULL AND parent_id IS NULL
            ORDER BY created_at DESC
          `;
        } else {
          rows = await this.sql`
            SELECT * FROM pages
            WHERE workspace_id = ${workspaceId} AND database_id IS NULL AND parent_id IS NULL AND is_archived = false
            ORDER BY created_at DESC
          `;
        }
      } else {
        if (options?.includeArchived) {
          rows = await this.sql`
            SELECT * FROM pages
            WHERE workspace_id = ${workspaceId} AND database_id IS NULL AND parent_id = ${options.parentId}
            ORDER BY created_at DESC
          `;
        } else {
          rows = await this.sql`
            SELECT * FROM pages
            WHERE workspace_id = ${workspaceId} AND database_id IS NULL AND parent_id = ${options.parentId} AND is_archived = false
            ORDER BY created_at DESC
          `;
        }
      }
    } else {
      if (options?.includeArchived) {
        rows = await this.sql`
          SELECT * FROM pages
          WHERE workspace_id = ${workspaceId} AND database_id IS NULL
          ORDER BY created_at DESC
        `;
      } else {
        rows = await this.sql`
          SELECT * FROM pages
          WHERE workspace_id = ${workspaceId} AND database_id IS NULL AND is_archived = false
          ORDER BY created_at DESC
        `;
      }
    }
    return rows.map((row) => this.mapRow(row));
  }

  async listByParent(parentId: string, parentType: string): Promise<Page[]> {
    const rows = await this.sql`
      SELECT * FROM pages
      WHERE parent_id = ${parentId} AND parent_type = ${parentType} AND is_archived = false
      ORDER BY created_at DESC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async listByDatabase(
    databaseId: string,
    options?: { cursor?: string; limit?: number; filter?: FilterGroup; sorts?: Sort[] }
  ): Promise<PaginatedResult<Page>> {
    const limit = options?.limit ?? 50;

    // Fetch all items for filtering/sorting (can be optimized with query-level filtering)
    const rows = await this.sql`
      SELECT * FROM pages WHERE database_id = ${databaseId} AND is_archived = false
    `;

    let items = rows.map((row) => this.mapRow(row));

    // Apply filter in memory
    if (options?.filter) {
      items = items.filter((item) => this.matchesFilter(item, options.filter!));
    }

    // Apply sorts in memory
    if (options?.sorts && options.sorts.length > 0) {
      items = this.applySorts(items, options.sorts);
    } else {
      items.sort((a, b) => (a.rowPosition ?? 0) - (b.rowPosition ?? 0));
    }

    // Apply cursor and pagination
    if (options?.cursor) {
      const cursorPos = parseFloat(options.cursor);
      const cursorIndex = items.findIndex((item) => (item.rowPosition ?? 0) > cursorPos);
      items = cursorIndex >= 0 ? items.slice(cursorIndex) : [];
    }

    const hasMore = items.length > limit;
    items = items.slice(0, limit);
    const nextCursor = hasMore ? String(items[items.length - 1]?.rowPosition ?? 0) : undefined;

    return { items, nextCursor, hasMore };
  }

  private matchesFilter(item: Page, filter: FilterGroup): boolean {
    const results = filter.conditions.map((condition) => {
      if ('type' in condition) {
        return this.matchesFilter(item, condition as FilterGroup);
      }
      return this.matchesCondition(item, condition);
    });
    return filter.type === 'and' ? results.every(Boolean) : results.some(Boolean);
  }

  private matchesCondition(
    item: Page,
    condition: { propertyId: string; operator: string; value?: unknown }
  ): boolean {
    const value = item.properties?.[condition.propertyId];
    const target = condition.value;

    switch (condition.operator) {
      case 'equals':
        return value === target;
      case 'not_equals':
        return value !== target;
      case 'contains':
        return typeof value === 'string' && typeof target === 'string' && value.includes(target);
      case 'not_contains':
        return typeof value === 'string' && typeof target === 'string' && !value.includes(target);
      case 'is_empty':
        return value === null || value === undefined || value === '';
      case 'is_not_empty':
        return value !== null && value !== undefined && value !== '';
      default:
        return true;
    }
  }

  private applySorts(items: Page[], sorts: Sort[]): Page[] {
    return [...items].sort((a, b) => {
      for (const sort of sorts) {
        const aVal = a.properties?.[sort.propertyId];
        const bVal = b.properties?.[sort.propertyId];

        let cmp = 0;
        if (aVal == null && bVal == null) cmp = 0;
        else if (aVal == null) cmp = 1;
        else if (bVal == null) cmp = -1;
        else if (typeof aVal === 'string' && typeof bVal === 'string') {
          cmp = aVal.localeCompare(bVal);
        } else if (typeof aVal === 'number' && typeof bVal === 'number') {
          cmp = aVal - bVal;
        } else {
          cmp = String(aVal).localeCompare(String(bVal));
        }

        if (cmp !== 0) {
          return sort.direction === 'descending' ? -cmp : cmp;
        }
      }
      return 0;
    });
  }

  async update(id: string, data: Partial<Page>): Promise<Page> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.title !== undefined) {
      updates.push('title');
      values.push(data.title);
    }
    if (data.icon !== undefined) {
      updates.push('icon');
      values.push(data.icon);
    }
    if (data.cover !== undefined) {
      updates.push('cover');
      values.push(data.cover);
    }
    if (data.coverY !== undefined) {
      updates.push('cover_y');
      values.push(data.coverY);
    }
    if (data.properties !== undefined) {
      updates.push('properties');
      values.push(JSON.stringify(data.properties));
    }
    if (data.isArchived !== undefined) {
      updates.push('is_archived');
      values.push(data.isArchived);
    }
    if (data.parentId !== undefined) {
      updates.push('parent_id');
      values.push(data.parentId);
    }
    if (data.parentType !== undefined) {
      updates.push('parent_type');
      values.push(data.parentType);
    }
    if (data.rowPosition !== undefined) {
      updates.push('row_position');
      values.push(data.rowPosition);
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE pages SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const page = await this.getById(id);
    if (!page) throw new Error('Page not found');
    return page;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM pages WHERE id = ${id}`;
  }

  async getHierarchy(id: string): Promise<{ id: string; title: string; icon?: string | null }[]> {
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
      properties: typeof row.properties === 'string' ? JSON.parse(row.properties) : (row.properties ?? {}),
      isTemplate: Boolean(row.is_template),
      isArchived: Boolean(row.is_archived),
      createdBy: row.created_by as string,
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Block Store
class PostgresBlockStore implements BlockStore {
  constructor(private sql: postgres.Sql) {}

  async create(block: Omit<Block, 'createdAt' | 'updatedAt'>): Promise<Block> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO blocks (id, page_id, parent_id, type, content, position, created_at, updated_at)
      VALUES (${block.id}, ${block.pageId}, ${block.parentId ?? null}, ${block.type},
        ${JSON.stringify(block.content)}, ${block.position}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Block | null> {
    const [row] = await this.sql`SELECT * FROM blocks WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async listByPage(pageId: string): Promise<Block[]> {
    const rows = await this.sql`
      SELECT * FROM blocks WHERE page_id = ${pageId} ORDER BY position ASC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async listByParent(parentId: string): Promise<Block[]> {
    const rows = await this.sql`
      SELECT * FROM blocks WHERE parent_id = ${parentId} ORDER BY position ASC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Block>): Promise<Block> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.type !== undefined) {
      updates.push('type');
      values.push(data.type);
    }
    if (data.content !== undefined) {
      updates.push('content');
      values.push(JSON.stringify(data.content));
    }
    if (data.position !== undefined) {
      updates.push('position');
      values.push(data.position);
    }
    if (data.parentId !== undefined) {
      updates.push('parent_id');
      values.push(data.parentId);
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE blocks SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const block = await this.getById(id);
    if (!block) throw new Error('Block not found');
    return block;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM blocks WHERE id = ${id}`;
  }

  async deleteByPage(pageId: string): Promise<void> {
    await this.sql`DELETE FROM blocks WHERE page_id = ${pageId}`;
  }

  async getMaxPosition(pageId: string, parentId?: string): Promise<number> {
    let row;
    if (parentId) {
      [row] = await this.sql`
        SELECT COALESCE(MAX(position), 0) as max FROM blocks WHERE page_id = ${pageId} AND parent_id = ${parentId}
      `;
    } else {
      [row] = await this.sql`
        SELECT COALESCE(MAX(position), 0) as max FROM blocks WHERE page_id = ${pageId} AND parent_id IS NULL
      `;
    }
    return ((row?.max as number) ?? 0) + 1;
  }

  async batchUpsert(pageId: string, blocks: Block[]): Promise<void> {
    await this.sql.begin(async (tx) => {
      // Delete existing blocks for the page
      await tx`DELETE FROM blocks WHERE page_id = ${pageId}`;

      // Insert new blocks
      for (const block of blocks) {
        await tx`
          INSERT INTO blocks (id, page_id, parent_id, type, content, position, created_at, updated_at)
          VALUES (${block.id}, ${block.pageId}, ${block.parentId ?? null}, ${block.type},
            ${JSON.stringify(block.content)}, ${block.position}, ${block.createdAt}, ${block.updatedAt})
        `;
      }
    });
  }

  private mapRow(row: Record<string, unknown>): Block {
    return {
      id: row.id as string,
      pageId: row.page_id as string,
      parentId: row.parent_id as string | undefined,
      type: row.type as Block['type'],
      content: typeof row.content === 'string' ? JSON.parse(row.content) : (row.content ?? {}),
      position: row.position as number,
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Database Store
class PostgresDatabaseStore implements DatabaseStore {
  constructor(private sql: postgres.Sql) {}

  async create(database: Omit<DbModel, 'createdAt' | 'updatedAt'>): Promise<DbModel> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO databases (id, workspace_id, page_id, title, icon, cover, is_inline, properties, created_at, updated_at)
      VALUES (${database.id}, ${database.workspaceId}, ${database.pageId}, ${database.title},
        ${database.icon ?? null}, ${database.cover ?? null}, ${database.isInline},
        ${JSON.stringify(database.properties)}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<DbModel | null> {
    const [row] = await this.sql`SELECT * FROM databases WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async getByPageId(pageId: string): Promise<DbModel | null> {
    const [row] = await this.sql`SELECT * FROM databases WHERE page_id = ${pageId}`;
    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(workspaceId: string): Promise<DbModel[]> {
    const rows = await this.sql`SELECT * FROM databases WHERE workspace_id = ${workspaceId}`;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<DbModel>): Promise<DbModel> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.title !== undefined) {
      updates.push('title');
      values.push(data.title);
    }
    if (data.icon !== undefined) {
      updates.push('icon');
      values.push(data.icon);
    }
    if (data.cover !== undefined) {
      updates.push('cover');
      values.push(data.cover);
    }
    if (data.properties !== undefined) {
      updates.push('properties');
      values.push(JSON.stringify(data.properties));
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE databases SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const database = await this.getById(id);
    if (!database) throw new Error('Database not found');
    return database;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM databases WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): DbModel {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      pageId: row.page_id as string,
      title: row.title as string,
      icon: row.icon as string | undefined,
      cover: row.cover as string | undefined,
      isInline: Boolean(row.is_inline),
      properties: typeof row.properties === 'string' ? JSON.parse(row.properties) : (row.properties ?? []),
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL View Store
class PostgresViewStore implements ViewStore {
  constructor(private sql: postgres.Sql) {}

  async create(view: Omit<View, 'createdAt' | 'updatedAt'>): Promise<View> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO views (id, database_id, name, type, filter, sorts, properties, group_by, calendar_by, position, created_at, updated_at)
      VALUES (${view.id}, ${view.databaseId}, ${view.name}, ${view.type},
        ${view.filter ? JSON.stringify(view.filter) : null}, ${view.sorts ? JSON.stringify(view.sorts) : null},
        ${view.properties ? JSON.stringify(view.properties) : null}, ${view.groupBy ?? null},
        ${view.calendarBy ?? null}, ${view.position}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<View | null> {
    const [row] = await this.sql`SELECT * FROM views WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async listByDatabase(databaseId: string): Promise<View[]> {
    const rows = await this.sql`
      SELECT * FROM views WHERE database_id = ${databaseId} ORDER BY position ASC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<View>): Promise<View> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name');
      values.push(data.name);
    }
    if (data.type !== undefined) {
      updates.push('type');
      values.push(data.type);
    }
    if (data.filter !== undefined) {
      updates.push('filter');
      values.push(data.filter ? JSON.stringify(data.filter) : null);
    }
    if (data.sorts !== undefined) {
      updates.push('sorts');
      values.push(data.sorts ? JSON.stringify(data.sorts) : null);
    }
    if (data.properties !== undefined) {
      updates.push('properties');
      values.push(data.properties ? JSON.stringify(data.properties) : null);
    }
    if (data.groupBy !== undefined) {
      updates.push('group_by');
      values.push(data.groupBy);
    }
    if (data.calendarBy !== undefined) {
      updates.push('calendar_by');
      values.push(data.calendarBy);
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE views SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const view = await this.getById(id);
    if (!view) throw new Error('View not found');
    return view;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM views WHERE id = ${id}`;
  }

  async getMaxPosition(databaseId: string): Promise<number> {
    const [row] = await this.sql`
      SELECT COALESCE(MAX(position), 0) as max FROM views WHERE database_id = ${databaseId}
    `;
    return ((row?.max as number) ?? 0) + 1;
  }

  private mapRow(row: Record<string, unknown>): View {
    return {
      id: row.id as string,
      databaseId: row.database_id as string,
      name: row.name as string,
      type: row.type as View['type'],
      filter: row.filter ? (typeof row.filter === 'string' ? JSON.parse(row.filter) : row.filter) : null,
      sorts: row.sorts ? (typeof row.sorts === 'string' ? JSON.parse(row.sorts) : row.sorts) : null,
      properties: row.properties
        ? typeof row.properties === 'string'
          ? JSON.parse(row.properties)
          : row.properties
        : null,
      groupBy: row.group_by as string | undefined,
      calendarBy: row.calendar_by as string | undefined,
      position: row.position as number,
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Comment Store
class PostgresCommentStore implements CommentStore {
  constructor(private sql: postgres.Sql) {}

  async create(comment: Omit<Comment, 'createdAt' | 'updatedAt'>): Promise<Comment> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO comments (id, workspace_id, target_type, target_id, parent_id, content, author_id, is_resolved, created_at, updated_at)
      VALUES (${comment.id}, ${comment.workspaceId}, ${comment.targetType}, ${comment.targetId},
        ${comment.parentId ?? null}, ${JSON.stringify(comment.content)}, ${comment.authorId},
        ${comment.isResolved}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Comment | null> {
    const [row] = await this.sql`SELECT * FROM comments WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async listByTarget(targetType: string, targetId: string): Promise<Comment[]> {
    const rows = await this.sql`
      SELECT * FROM comments WHERE target_type = ${targetType} AND target_id = ${targetId}
      ORDER BY created_at ASC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async listByPage(pageId: string): Promise<Comment[]> {
    const rows = await this.sql`
      SELECT * FROM comments WHERE target_type = 'page' AND target_id = ${pageId}
      ORDER BY created_at ASC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Comment>): Promise<Comment> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.content !== undefined) {
      updates.push('content');
      values.push(JSON.stringify(data.content));
    }
    if (data.isResolved !== undefined) {
      updates.push('is_resolved');
      values.push(data.isResolved);
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE comments SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const comment = await this.getById(id);
    if (!comment) throw new Error('Comment not found');
    return comment;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM comments WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): Comment {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      targetType: row.target_type as Comment['targetType'],
      targetId: row.target_id as string,
      parentId: row.parent_id as string | undefined,
      content: typeof row.content === 'string' ? JSON.parse(row.content) : (row.content ?? []),
      authorId: row.author_id as string,
      isResolved: Boolean(row.is_resolved),
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Share Store
class PostgresShareStore implements ShareStore {
  constructor(private sql: postgres.Sql) {}

  async create(share: Omit<Share, 'createdAt' | 'updatedAt'>): Promise<Share> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO shares (id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at, updated_at)
      VALUES (${share.id}, ${share.pageId}, ${share.type}, ${share.permission},
        ${share.userId ?? null}, ${share.token ?? null}, ${share.password ?? null},
        ${share.expiresAt ?? null}, ${share.domain ?? null}, ${share.createdBy}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Share | null> {
    const [row] = await this.sql`SELECT * FROM shares WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async getByToken(token: string): Promise<Share | null> {
    const [row] = await this.sql`SELECT * FROM shares WHERE token = ${token}`;
    return row ? this.mapRow(row) : null;
  }

  async listByPage(pageId: string): Promise<Share[]> {
    const rows = await this.sql`SELECT * FROM shares WHERE page_id = ${pageId}`;
    return rows.map((row) => this.mapRow(row));
  }

  async update(id: string, data: Partial<Share>): Promise<Share> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.permission !== undefined) {
      updates.push('permission');
      values.push(data.permission);
    }
    if (data.password !== undefined) {
      updates.push('password');
      values.push(data.password);
    }
    if (data.expiresAt !== undefined) {
      updates.push('expires_at');
      values.push(data.expiresAt);
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE shares SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const share = await this.getById(id);
    if (!share) throw new Error('Share not found');
    return share;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM shares WHERE id = ${id}`;
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
      expiresAt: row.expires_at ? toISO(row.expires_at as Date) : undefined,
      domain: row.domain as string | undefined,
      createdBy: row.created_by as string,
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Favorite Store
class PostgresFavoriteStore implements FavoriteStore {
  constructor(private sql: postgres.Sql) {}

  async create(favorite: Omit<Favorite, 'createdAt'>): Promise<Favorite> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO favorites (id, user_id, page_id, position, created_at)
      VALUES (${favorite.id}, ${favorite.userId}, ${favorite.pageId}, ${favorite.position}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Favorite | null> {
    const [row] = await this.sql`SELECT * FROM favorites WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async getByUserAndPage(userId: string, pageId: string): Promise<Favorite | null> {
    const [row] = await this.sql`
      SELECT * FROM favorites WHERE user_id = ${userId} AND page_id = ${pageId}
    `;
    return row ? this.mapRow(row) : null;
  }

  async listByUser(userId: string, workspaceId: string): Promise<Favorite[]> {
    const rows = await this.sql`
      SELECT f.* FROM favorites f
      INNER JOIN pages p ON p.id = f.page_id
      WHERE f.user_id = ${userId} AND p.workspace_id = ${workspaceId}
      ORDER BY f.position ASC
    `;
    return rows.map((row) => this.mapRow(row));
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM favorites WHERE id = ${id}`;
  }

  async deleteByUserAndPage(userId: string, pageId: string): Promise<void> {
    await this.sql`DELETE FROM favorites WHERE user_id = ${userId} AND page_id = ${pageId}`;
  }

  async getMaxPosition(userId: string): Promise<number> {
    const [row] = await this.sql`
      SELECT COALESCE(MAX(position), 0) as max FROM favorites WHERE user_id = ${userId}
    `;
    return ((row?.max as number) ?? 0) + 1;
  }

  private mapRow(row: Record<string, unknown>): Favorite {
    return {
      id: row.id as string,
      userId: row.user_id as string,
      pageId: row.page_id as string,
      position: row.position as number,
      createdAt: toISO(row.created_at as Date),
    };
  }
}

// PostgreSQL Activity Store
class PostgresActivityStore implements ActivityStore {
  constructor(private sql: postgres.Sql) {}

  async create(activity: Omit<Activity, 'createdAt'>): Promise<Activity> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO activities (id, workspace_id, user_id, action, target_type, target_id, metadata, created_at)
      VALUES (${activity.id}, ${activity.workspaceId}, ${activity.userId}, ${activity.action},
        ${activity.targetType}, ${activity.targetId}, ${JSON.stringify(activity.metadata)}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async listByWorkspace(
    workspaceId: string,
    options?: { cursor?: string; limit?: number }
  ): Promise<PaginatedResult<Activity>> {
    const limit = options?.limit ?? 50;
    let rows;

    if (options?.cursor) {
      rows = await this.sql`
        SELECT * FROM activities
        WHERE workspace_id = ${workspaceId} AND created_at < ${options.cursor}
        ORDER BY created_at DESC
        LIMIT ${limit + 1}
      `;
    } else {
      rows = await this.sql`
        SELECT * FROM activities
        WHERE workspace_id = ${workspaceId}
        ORDER BY created_at DESC
        LIMIT ${limit + 1}
      `;
    }

    const items = rows.slice(0, limit).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
    const nextCursor = hasMore ? items[items.length - 1]?.createdAt : undefined;

    return { items, nextCursor, hasMore };
  }

  async listByUser(
    userId: string,
    options?: { cursor?: string; limit?: number }
  ): Promise<PaginatedResult<Activity>> {
    const limit = options?.limit ?? 50;
    let rows;

    if (options?.cursor) {
      rows = await this.sql`
        SELECT * FROM activities
        WHERE user_id = ${userId} AND created_at < ${options.cursor}
        ORDER BY created_at DESC
        LIMIT ${limit + 1}
      `;
    } else {
      rows = await this.sql`
        SELECT * FROM activities
        WHERE user_id = ${userId}
        ORDER BY created_at DESC
        LIMIT ${limit + 1}
      `;
    }

    const items = rows.slice(0, limit).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
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
      metadata: typeof row.metadata === 'string' ? JSON.parse(row.metadata) : (row.metadata ?? {}),
      createdAt: toISO(row.created_at as Date),
    };
  }
}

// PostgreSQL Notification Store
class PostgresNotificationStore implements NotificationStore {
  constructor(private sql: postgres.Sql) {}

  async create(notification: Omit<Notification, 'createdAt'>): Promise<Notification> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO notifications (id, user_id, type, title, message, metadata, is_read, created_at)
      VALUES (${notification.id}, ${notification.userId}, ${notification.type}, ${notification.title},
        ${notification.message ?? null}, ${JSON.stringify(notification.metadata)}, ${notification.isRead}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<Notification | null> {
    const [row] = await this.sql`SELECT * FROM notifications WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async listByUser(
    userId: string,
    options?: { unreadOnly?: boolean; cursor?: string; limit?: number }
  ): Promise<PaginatedResult<Notification>> {
    const limit = options?.limit ?? 50;
    let rows;

    if (options?.unreadOnly) {
      if (options?.cursor) {
        rows = await this.sql`
          SELECT * FROM notifications
          WHERE user_id = ${userId} AND is_read = false AND created_at < ${options.cursor}
          ORDER BY created_at DESC
          LIMIT ${limit + 1}
        `;
      } else {
        rows = await this.sql`
          SELECT * FROM notifications
          WHERE user_id = ${userId} AND is_read = false
          ORDER BY created_at DESC
          LIMIT ${limit + 1}
        `;
      }
    } else {
      if (options?.cursor) {
        rows = await this.sql`
          SELECT * FROM notifications
          WHERE user_id = ${userId} AND created_at < ${options.cursor}
          ORDER BY created_at DESC
          LIMIT ${limit + 1}
        `;
      } else {
        rows = await this.sql`
          SELECT * FROM notifications
          WHERE user_id = ${userId}
          ORDER BY created_at DESC
          LIMIT ${limit + 1}
        `;
      }
    }

    const items = rows.slice(0, limit).map((row) => this.mapRow(row));
    const hasMore = rows.length > limit;
    const nextCursor = hasMore ? items[items.length - 1]?.createdAt : undefined;

    return { items, nextCursor, hasMore };
  }

  async markAsRead(id: string): Promise<void> {
    await this.sql`UPDATE notifications SET is_read = true WHERE id = ${id}`;
  }

  async markAllAsRead(userId: string): Promise<void> {
    await this.sql`UPDATE notifications SET is_read = true WHERE user_id = ${userId}`;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM notifications WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): Notification {
    return {
      id: row.id as string,
      userId: row.user_id as string,
      type: row.type as string,
      title: row.title as string,
      message: row.message as string | undefined,
      metadata: typeof row.metadata === 'string' ? JSON.parse(row.metadata) : (row.metadata ?? {}),
      isRead: Boolean(row.is_read),
      createdAt: toISO(row.created_at as Date),
    };
  }
}

// PostgreSQL Synced Block Store
class PostgresSyncedBlockStore implements SyncedBlockStore {
  constructor(private sql: postgres.Sql) {}

  async create(syncedBlock: Omit<SyncedBlock, 'createdAt' | 'updatedAt'>): Promise<SyncedBlock> {
    const now = nowISO();
    const [result] = await this.sql`
      INSERT INTO synced_blocks (id, workspace_id, source_block_id, content, created_by, created_at, updated_at)
      VALUES (${syncedBlock.id}, ${syncedBlock.workspaceId}, ${syncedBlock.sourceBlockId},
        ${JSON.stringify(syncedBlock.content)}, ${syncedBlock.createdBy}, ${now}, ${now})
      RETURNING *
    `;
    return this.mapRow(result);
  }

  async getById(id: string): Promise<SyncedBlock | null> {
    const [row] = await this.sql`SELECT * FROM synced_blocks WHERE id = ${id}`;
    return row ? this.mapRow(row) : null;
  }

  async listByWorkspace(workspaceId: string): Promise<SyncedBlock[]> {
    const rows = await this.sql`SELECT * FROM synced_blocks WHERE workspace_id = ${workspaceId}`;
    return rows.map((row) => this.mapRow(row));
  }

  async listByPage(_pageId: string): Promise<SyncedBlock[]> {
    // Not implemented - would need to join with blocks table
    return [];
  }

  async update(id: string, data: Partial<SyncedBlock>): Promise<SyncedBlock> {
    const now = nowISO();
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.content !== undefined) {
      updates.push('content');
      values.push(JSON.stringify(data.content));
    }

    if (updates.length > 0) {
      const setClause = updates.map((col, i) => `${col} = $${i + 2}`).join(', ');
      await this.sql.unsafe(
        `UPDATE synced_blocks SET ${setClause}, updated_at = $1 WHERE id = $${updates.length + 2}`,
        [now, ...values, id]
      );
    }

    const syncedBlock = await this.getById(id);
    if (!syncedBlock) throw new Error('Synced block not found');
    return syncedBlock;
  }

  async delete(id: string): Promise<void> {
    await this.sql`DELETE FROM synced_blocks WHERE id = ${id}`;
  }

  private mapRow(row: Record<string, unknown>): SyncedBlock {
    return {
      id: row.id as string,
      workspaceId: row.workspace_id as string,
      sourceBlockId: row.source_block_id as string,
      content: typeof row.content === 'string' ? JSON.parse(row.content) : (row.content ?? null),
      createdBy: row.created_by as string,
      createdAt: toISO(row.created_at as Date),
      updatedAt: toISO(row.updated_at as Date),
    };
  }
}

// PostgreSQL Schema
const SCHEMA_SQL = `
-- Users & Authentication
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  avatar_url TEXT,
  password_hash TEXT NOT NULL,
  settings JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Workspaces
CREATE TABLE IF NOT EXISTS workspaces (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  icon TEXT,
  domain TEXT,
  plan TEXT DEFAULT 'free',
  settings JSONB DEFAULT '{}',
  owner_id TEXT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS members (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(workspace_id, user_id)
);

-- Pages
CREATE TABLE IF NOT EXISTS pages (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  parent_id TEXT,
  parent_type TEXT NOT NULL DEFAULT 'workspace',
  database_id TEXT,
  row_position DOUBLE PRECISION,
  title TEXT NOT NULL DEFAULT '',
  icon TEXT,
  cover TEXT,
  cover_y DOUBLE PRECISION DEFAULT 0.5,
  properties JSONB DEFAULT '{}',
  is_template BOOLEAN DEFAULT FALSE,
  is_archived BOOLEAN DEFAULT FALSE,
  created_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Blocks
CREATE TABLE IF NOT EXISTS blocks (
  id TEXT PRIMARY KEY,
  page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  parent_id TEXT,
  type TEXT NOT NULL,
  content JSONB DEFAULT '{}',
  position DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Databases
CREATE TABLE IF NOT EXISTS databases (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  icon TEXT,
  cover TEXT,
  is_inline BOOLEAN DEFAULT FALSE,
  properties JSONB DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Views
CREATE TABLE IF NOT EXISTS views (
  id TEXT PRIMARY KEY,
  database_id TEXT NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'table',
  filter JSONB,
  sorts JSONB,
  properties JSONB,
  group_by TEXT,
  calendar_by TEXT,
  position DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  parent_id TEXT,
  content JSONB NOT NULL,
  author_id TEXT NOT NULL REFERENCES users(id),
  is_resolved BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Shares
CREATE TABLE IF NOT EXISTS shares (
  id TEXT PRIMARY KEY,
  page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  permission TEXT NOT NULL DEFAULT 'read',
  user_id TEXT,
  token TEXT,
  password TEXT,
  expires_at TIMESTAMPTZ,
  domain TEXT,
  created_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Favorites
CREATE TABLE IF NOT EXISTS favorites (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
  position DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, page_id)
);

-- Synced Blocks
CREATE TABLE IF NOT EXISTS synced_blocks (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  source_block_id TEXT NOT NULL,
  content JSONB NOT NULL,
  created_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Activity Log
CREATE TABLE IF NOT EXISTS activities (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id),
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT,
  metadata JSONB DEFAULT '{}',
  is_read BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance Indexes
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
CREATE INDEX IF NOT EXISTS idx_databases_page_id ON databases(page_id);
CREATE INDEX IF NOT EXISTS idx_views_database_id ON views(database_id);
CREATE INDEX IF NOT EXISTS idx_comments_target ON comments(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_comments_workspace ON comments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_shares_page_id ON shares(page_id);
CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token);
CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_activities_workspace_id ON activities(workspace_id);
CREATE INDEX IF NOT EXISTS idx_activities_created_at ON activities(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
`;

// Main PostgreSQL Store
export class PostgresStore implements Store {
  private sql!: postgres.Sql;
  private schemaName?: string;
  users!: UserStore;
  sessions!: SessionStore;
  workspaces!: WorkspaceStore;
  members!: MemberStore;
  pages!: PageStore;
  blocks!: BlockStore;
  databases!: DatabaseStore;
  views!: ViewStore;
  comments!: CommentStore;
  shares!: ShareStore;
  favorites!: FavoriteStore;
  activities!: ActivityStore;
  notifications!: NotificationStore;
  syncedBlocks!: SyncedBlockStore;

  constructor(
    private connectionUrl: string,
    options?: { schema?: string }
  ) {
    this.schemaName = options?.schema;
  }

  async init(): Promise<void> {
    // Dynamically import postgres
    if (!Postgres) {
      Postgres = (await import('postgres')).default;
    }

    // For test isolation (with schema), use single connection to preserve SET search_path
    // This is needed for connection poolers like Neon that don't support startup params
    const maxConnections = this.schemaName ? 1 : 10;

    // Create connection
    this.sql = Postgres(this.connectionUrl, {
      max: maxConnections,
      idle_timeout: 30, // Close idle connections after 30s
      connect_timeout: 10, // Connection timeout
    });

    // If schema is specified (for test isolation), create and use it
    if (this.schemaName) {
      // Create schema and set search_path
      await this.sql.unsafe(`CREATE SCHEMA IF NOT EXISTS "${this.schemaName}"`);
      // Set search_path - with max: 1 this will persist for all subsequent queries
      await this.sql.unsafe(`SET search_path TO "${this.schemaName}"`);
    }

    // Create all tables
    await this.sql.unsafe(SCHEMA_SQL);

    // Initialize stores
    this.users = new PostgresUserStore(this.sql);
    this.sessions = new PostgresSessionStore(this.sql);
    this.workspaces = new PostgresWorkspaceStore(this.sql);
    this.members = new PostgresMemberStore(this.sql);
    this.pages = new PostgresPageStore(this.sql);
    this.blocks = new PostgresBlockStore(this.sql);
    this.databases = new PostgresDatabaseStore(this.sql);
    this.views = new PostgresViewStore(this.sql);
    this.comments = new PostgresCommentStore(this.sql);
    this.shares = new PostgresShareStore(this.sql);
    this.favorites = new PostgresFavoriteStore(this.sql);
    this.activities = new PostgresActivityStore(this.sql);
    this.notifications = new PostgresNotificationStore(this.sql);
    this.syncedBlocks = new PostgresSyncedBlockStore(this.sql);
  }

  async close(): Promise<void> {
    // Drop schema if it was created for test isolation
    if (this.schemaName) {
      try {
        await this.sql.unsafe(`DROP SCHEMA IF EXISTS "${this.schemaName}" CASCADE`);
      } catch {
        // Ignore errors during cleanup
      }
    }
    await this.sql.end();
  }
}

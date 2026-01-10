/**
 * SQLite database driver using better-sqlite3
 */

import BetterSqlite3 from 'better-sqlite3';
import { ulid } from 'ulid';
import { SCHEMA_SQL } from './schema.js';
import type {
  Database,
  DbRow,
  RecordWithFields,
  RecordQueryOptions,
  User,
  CreateUserInput,
  Session,
  CreateSessionInput,
  Workspace,
  CreateWorkspaceInput,
  UpdateWorkspaceInput,
  Base,
  CreateBaseInput,
  UpdateBaseInput,
  Table,
  CreateTableInput,
  UpdateTableInput,
  Field,
  CreateFieldInput,
  UpdateFieldInput,
  SelectOption,
  CreateSelectOptionInput,
  Record,
  CellValue,
  View,
  CreateViewInput,
  UpdateViewInput,
  Comment,
  CreateCommentInput,
  UpdateCommentInput,
  Share,
  CreateShareInput,
} from './types.js';

/**
 * SQLite database implementation
 */
export class SqliteDriver implements Database {
  private db: BetterSqlite3.Database;

  constructor(dbPath: string) {
    this.db = new BetterSqlite3(dbPath);
    this.db.pragma('journal_mode = WAL');
    this.db.pragma('foreign_keys = ON');
  }

  static createInMemory(): SqliteDriver {
    const driver = new SqliteDriver(':memory:');
    return driver;
  }

  async ensure(): Promise<void> {
    this.db.exec(SCHEMA_SQL);
  }

  async close(): Promise<void> {
    this.db.close();
  }

  // ============================================================================
  // Users
  // ============================================================================

  async createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?)
    `);
    stmt.run(input.id, input.email, input.name, input.password_hash, now, now);

    return {
      id: input.id,
      email: input.email,
      name: input.name,
      password_hash: input.password_hash,
      created_at: now,
      updated_at: now,
    };
  }

  async getUserById(id: string): Promise<User | null> {
    const stmt = this.db.prepare('SELECT * FROM users WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToUser(row) : null;
  }

  async getUserByEmail(email: string): Promise<User | null> {
    const stmt = this.db.prepare('SELECT * FROM users WHERE email = ?');
    const row = stmt.get(email) as DbRow | undefined;
    return row ? this.rowToUser(row) : null;
  }

  private rowToUser(row: DbRow): User {
    return {
      id: row.id as string,
      email: row.email as string,
      name: row.name as string,
      avatar_url: row.avatar_url as string | null,
      password_hash: row.password_hash as string,
      settings: row.settings ? JSON.parse(row.settings as string) : {},
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Sessions
  // ============================================================================

  async createSession(input: CreateSessionInput): Promise<Session> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO sessions (id, user_id, expires_at, created_at)
      VALUES (?, ?, ?, ?)
    `);
    stmt.run(input.id, input.user_id, input.expires_at, now);

    return {
      id: input.id,
      user_id: input.user_id,
      expires_at: input.expires_at,
      created_at: now,
    };
  }

  async getSessionById(id: string): Promise<Session | null> {
    const stmt = this.db.prepare('SELECT * FROM sessions WHERE id = ? AND expires_at > datetime("now")');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToSession(row) : null;
  }

  async deleteSession(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM sessions WHERE id = ?');
    stmt.run(id);
  }

  async deleteExpiredSessions(): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM sessions WHERE expires_at <= datetime("now")');
    stmt.run();
  }

  private rowToSession(row: DbRow): Session {
    return {
      id: row.id as string,
      user_id: row.user_id as string,
      expires_at: row.expires_at as string,
      created_at: row.created_at as string,
    };
  }

  // ============================================================================
  // Workspaces
  // ============================================================================

  async createWorkspace(input: CreateWorkspaceInput & { id: string; owner_id: string }): Promise<Workspace> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO workspaces (id, name, slug, icon, owner_id, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(input.id, input.name, input.slug, input.icon || null, input.owner_id, now, now);

    // Add owner as workspace member
    const memberStmt = this.db.prepare(`
      INSERT INTO workspace_members (id, workspace_id, user_id, role, joined_at)
      VALUES (?, ?, ?, 'owner', ?)
    `);
    memberStmt.run(ulid(), input.id, input.owner_id, now);

    return {
      id: input.id,
      name: input.name,
      slug: input.slug,
      icon: input.icon || null,
      plan: 'free',
      settings: {},
      owner_id: input.owner_id,
      created_at: now,
      updated_at: now,
    };
  }

  async getWorkspace(id: string): Promise<Workspace | null> {
    const stmt = this.db.prepare('SELECT * FROM workspaces WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToWorkspace(row) : null;
  }

  async getWorkspaceBySlug(slug: string): Promise<Workspace | null> {
    const stmt = this.db.prepare('SELECT * FROM workspaces WHERE slug = ?');
    const row = stmt.get(slug) as DbRow | undefined;
    return row ? this.rowToWorkspace(row) : null;
  }

  async getWorkspacesByUser(userId: string): Promise<Workspace[]> {
    const stmt = this.db.prepare(`
      SELECT w.* FROM workspaces w
      INNER JOIN workspace_members wm ON w.id = wm.workspace_id
      WHERE wm.user_id = ?
      ORDER BY w.created_at DESC
    `);
    const rows = stmt.all(userId) as DbRow[];
    return rows.map(row => this.rowToWorkspace(row));
  }

  async updateWorkspace(id: string, data: UpdateWorkspaceInput): Promise<Workspace | null> {
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.icon !== undefined) {
      updates.push('icon = ?');
      values.push(data.icon);
    }
    if (data.settings !== undefined) {
      updates.push('settings = ?');
      values.push(JSON.stringify(data.settings));
    }

    if (updates.length === 0) return this.getWorkspace(id);

    updates.push('updated_at = ?');
    values.push(new Date().toISOString());
    values.push(id);

    const stmt = this.db.prepare(`UPDATE workspaces SET ${updates.join(', ')} WHERE id = ?`);
    stmt.run(...values);

    return this.getWorkspace(id);
  }

  async deleteWorkspace(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM workspaces WHERE id = ?');
    stmt.run(id);
  }

  private rowToWorkspace(row: DbRow): Workspace {
    return {
      id: row.id as string,
      name: row.name as string,
      slug: row.slug as string,
      icon: row.icon as string | null,
      plan: row.plan as string,
      settings: row.settings ? JSON.parse(row.settings as string) : {},
      owner_id: row.owner_id as string,
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Bases
  // ============================================================================

  async createBase(input: CreateBaseInput & { id: string; workspace_id: string; created_by: string }): Promise<Base> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO bases (id, workspace_id, name, description, icon, color, created_by, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      input.id,
      input.workspace_id,
      input.name,
      input.description || null,
      input.icon || null,
      input.color || '#2D7FF9',
      input.created_by,
      now,
      now
    );

    return {
      id: input.id,
      workspace_id: input.workspace_id,
      name: input.name,
      description: input.description || null,
      icon: input.icon || null,
      color: input.color || '#2D7FF9',
      settings: {},
      is_template: false,
      created_by: input.created_by,
      created_at: now,
      updated_at: now,
    };
  }

  async getBase(id: string): Promise<Base | null> {
    const stmt = this.db.prepare('SELECT * FROM bases WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToBase(row) : null;
  }

  async getBasesByWorkspace(workspaceId: string): Promise<Base[]> {
    const stmt = this.db.prepare('SELECT * FROM bases WHERE workspace_id = ? ORDER BY created_at DESC');
    const rows = stmt.all(workspaceId) as DbRow[];
    return rows.map(row => this.rowToBase(row));
  }

  async updateBase(id: string, data: UpdateBaseInput): Promise<Base | null> {
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.description !== undefined) {
      updates.push('description = ?');
      values.push(data.description);
    }
    if (data.icon !== undefined) {
      updates.push('icon = ?');
      values.push(data.icon);
    }
    if (data.color !== undefined) {
      updates.push('color = ?');
      values.push(data.color);
    }
    if (data.settings !== undefined) {
      updates.push('settings = ?');
      values.push(JSON.stringify(data.settings));
    }

    if (updates.length === 0) return this.getBase(id);

    updates.push('updated_at = ?');
    values.push(new Date().toISOString());
    values.push(id);

    const stmt = this.db.prepare(`UPDATE bases SET ${updates.join(', ')} WHERE id = ?`);
    stmt.run(...values);

    return this.getBase(id);
  }

  async deleteBase(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM bases WHERE id = ?');
    stmt.run(id);
  }

  private rowToBase(row: DbRow): Base {
    return {
      id: row.id as string,
      workspace_id: row.workspace_id as string,
      name: row.name as string,
      description: row.description as string | null,
      icon: row.icon as string | null,
      color: row.color as string,
      settings: row.settings ? JSON.parse(row.settings as string) : {},
      is_template: Boolean(row.is_template),
      created_by: row.created_by as string,
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Tables
  // ============================================================================

  async createTable(input: CreateTableInput & { id: string; base_id: string; created_by: string }): Promise<Table> {
    const now = new Date().toISOString();

    // Get max position
    const posStmt = this.db.prepare('SELECT COALESCE(MAX(position), -1) + 1 as next_pos FROM tables WHERE base_id = ?');
    const posRow = posStmt.get(input.base_id) as DbRow;
    const position = posRow.next_pos as number;

    const stmt = this.db.prepare(`
      INSERT INTO tables (id, base_id, name, description, icon, position, created_by, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      input.id,
      input.base_id,
      input.name,
      input.description || null,
      input.icon || null,
      position,
      input.created_by,
      now,
      now
    );

    // Create default primary field
    const fieldId = ulid();
    const fieldStmt = this.db.prepare(`
      INSERT INTO fields (id, table_id, name, type, position, is_primary, created_by, created_at, updated_at)
      VALUES (?, ?, 'Name', 'text', 0, 1, ?, ?, ?)
    `);
    fieldStmt.run(fieldId, input.id, input.created_by, now, now);

    // Update table with primary field
    const updateStmt = this.db.prepare('UPDATE tables SET primary_field_id = ? WHERE id = ?');
    updateStmt.run(fieldId, input.id);

    // Create default grid view
    const viewId = ulid();
    const viewStmt = this.db.prepare(`
      INSERT INTO views (id, table_id, name, type, position, is_default, created_by, created_at, updated_at)
      VALUES (?, ?, 'Grid view', 'grid', 0, 1, ?, ?, ?)
    `);
    viewStmt.run(viewId, input.id, input.created_by, now, now);

    return {
      id: input.id,
      base_id: input.base_id,
      name: input.name,
      description: input.description || null,
      icon: input.icon || null,
      position,
      primary_field_id: fieldId,
      settings: {},
      created_by: input.created_by,
      created_at: now,
      updated_at: now,
    };
  }

  async getTable(id: string): Promise<Table | null> {
    const stmt = this.db.prepare('SELECT * FROM tables WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToTable(row) : null;
  }

  async getTablesByBase(baseId: string): Promise<Table[]> {
    const stmt = this.db.prepare('SELECT * FROM tables WHERE base_id = ? ORDER BY position');
    const rows = stmt.all(baseId) as DbRow[];
    return rows.map(row => this.rowToTable(row));
  }

  async updateTable(id: string, data: UpdateTableInput): Promise<Table | null> {
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.description !== undefined) {
      updates.push('description = ?');
      values.push(data.description);
    }
    if (data.icon !== undefined) {
      updates.push('icon = ?');
      values.push(data.icon);
    }
    if (data.settings !== undefined) {
      updates.push('settings = ?');
      values.push(JSON.stringify(data.settings));
    }

    if (updates.length === 0) return this.getTable(id);

    updates.push('updated_at = ?');
    values.push(new Date().toISOString());
    values.push(id);

    const stmt = this.db.prepare(`UPDATE tables SET ${updates.join(', ')} WHERE id = ?`);
    stmt.run(...values);

    return this.getTable(id);
  }

  async deleteTable(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM tables WHERE id = ?');
    stmt.run(id);
  }

  async reorderTables(baseId: string, tableIds: string[]): Promise<void> {
    const stmt = this.db.prepare('UPDATE tables SET position = ? WHERE id = ? AND base_id = ?');
    tableIds.forEach((tableId, index) => {
      stmt.run(index, tableId, baseId);
    });
  }

  private rowToTable(row: DbRow): Table {
    return {
      id: row.id as string,
      base_id: row.base_id as string,
      name: row.name as string,
      description: row.description as string | null,
      icon: row.icon as string | null,
      position: row.position as number,
      primary_field_id: row.primary_field_id as string | null,
      settings: row.settings ? JSON.parse(row.settings as string) : {},
      created_by: row.created_by as string,
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Fields
  // ============================================================================

  async createField(input: CreateFieldInput & { id: string; table_id: string; created_by: string; position: number }): Promise<Field> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO fields (id, table_id, name, type, description, options, position, created_by, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      input.id,
      input.table_id,
      input.name,
      input.type,
      input.description || null,
      JSON.stringify(input.options || {}),
      input.position,
      input.created_by,
      now,
      now
    );

    return {
      id: input.id,
      table_id: input.table_id,
      name: input.name,
      type: input.type,
      description: input.description || null,
      options: input.options || {},
      position: input.position,
      is_primary: false,
      is_computed: false,
      is_hidden: false,
      width: 200,
      created_by: input.created_by,
      created_at: now,
      updated_at: now,
    };
  }

  async getField(id: string): Promise<Field | null> {
    const stmt = this.db.prepare('SELECT * FROM fields WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToField(row) : null;
  }

  async getFieldsByTable(tableId: string): Promise<Field[]> {
    const stmt = this.db.prepare('SELECT * FROM fields WHERE table_id = ? ORDER BY position');
    const rows = stmt.all(tableId) as DbRow[];
    return rows.map(row => this.rowToField(row));
  }

  async updateField(id: string, data: UpdateFieldInput): Promise<Field | null> {
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.description !== undefined) {
      updates.push('description = ?');
      values.push(data.description);
    }
    if (data.options !== undefined) {
      updates.push('options = ?');
      values.push(JSON.stringify(data.options));
    }
    if (data.is_hidden !== undefined) {
      updates.push('is_hidden = ?');
      values.push(data.is_hidden ? 1 : 0);
    }
    if (data.width !== undefined) {
      updates.push('width = ?');
      values.push(data.width);
    }

    if (updates.length === 0) return this.getField(id);

    updates.push('updated_at = ?');
    values.push(new Date().toISOString());
    values.push(id);

    const stmt = this.db.prepare(`UPDATE fields SET ${updates.join(', ')} WHERE id = ?`);
    stmt.run(...values);

    return this.getField(id);
  }

  async deleteField(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM fields WHERE id = ?');
    stmt.run(id);
  }

  async reorderFields(tableId: string, fieldIds: string[]): Promise<void> {
    const stmt = this.db.prepare('UPDATE fields SET position = ? WHERE id = ? AND table_id = ?');
    fieldIds.forEach((fieldId, index) => {
      stmt.run(index, fieldId, tableId);
    });
  }

  async getMaxFieldPosition(tableId: string): Promise<number> {
    const stmt = this.db.prepare('SELECT COALESCE(MAX(position), -1) as max_pos FROM fields WHERE table_id = ?');
    const row = stmt.get(tableId) as DbRow;
    return row.max_pos as number;
  }

  private rowToField(row: DbRow): Field {
    return {
      id: row.id as string,
      table_id: row.table_id as string,
      name: row.name as string,
      type: row.type as Field['type'],
      description: row.description as string | null,
      options: row.options ? JSON.parse(row.options as string) : {},
      position: row.position as number,
      is_primary: Boolean(row.is_primary),
      is_computed: Boolean(row.is_computed),
      is_hidden: Boolean(row.is_hidden),
      width: row.width as number,
      created_by: row.created_by as string,
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Select Options
  // ============================================================================

  async createSelectOption(input: CreateSelectOptionInput & { id: string; field_id: string; position: number }): Promise<SelectOption> {
    const stmt = this.db.prepare(`
      INSERT INTO select_options (id, field_id, name, color, position)
      VALUES (?, ?, ?, ?, ?)
    `);
    stmt.run(input.id, input.field_id, input.name, input.color || '#CFDFFF', input.position);

    return {
      id: input.id,
      field_id: input.field_id,
      name: input.name,
      color: input.color || '#CFDFFF',
      position: input.position,
    };
  }

  async getSelectOptionsByField(fieldId: string): Promise<SelectOption[]> {
    const stmt = this.db.prepare('SELECT * FROM select_options WHERE field_id = ? ORDER BY position');
    const rows = stmt.all(fieldId) as DbRow[];
    return rows.map(row => ({
      id: row.id as string,
      field_id: row.field_id as string,
      name: row.name as string,
      color: row.color as string,
      position: row.position as number,
    }));
  }

  async updateSelectOption(id: string, name: string, color: string): Promise<SelectOption | null> {
    const stmt = this.db.prepare('UPDATE select_options SET name = ?, color = ? WHERE id = ?');
    stmt.run(name, color, id);

    const getStmt = this.db.prepare('SELECT * FROM select_options WHERE id = ?');
    const row = getStmt.get(id) as DbRow | undefined;
    if (!row) return null;

    return {
      id: row.id as string,
      field_id: row.field_id as string,
      name: row.name as string,
      color: row.color as string,
      position: row.position as number,
    };
  }

  async deleteSelectOption(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM select_options WHERE id = ?');
    stmt.run(id);
  }

  async reorderSelectOptions(fieldId: string, optionIds: string[]): Promise<void> {
    const stmt = this.db.prepare('UPDATE select_options SET position = ? WHERE id = ? AND field_id = ?');
    optionIds.forEach((optionId, index) => {
      stmt.run(index, optionId, fieldId);
    });
  }

  // ============================================================================
  // Records
  // ============================================================================

  async createRecord(input: { id: string; table_id: string; created_by: string; position: number }): Promise<Record> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO records (id, table_id, position, created_by, updated_by, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(input.id, input.table_id, input.position, input.created_by, input.created_by, now, now);

    return {
      id: input.id,
      table_id: input.table_id,
      position: input.position,
      created_by: input.created_by,
      created_at: now,
      updated_by: input.created_by,
      updated_at: now,
    };
  }

  async getRecord(id: string): Promise<Record | null> {
    const stmt = this.db.prepare('SELECT * FROM records WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToRecord(row) : null;
  }

  async getRecordWithFields(id: string): Promise<RecordWithFields | null> {
    const record = await this.getRecord(id);
    if (!record) return null;

    const cells = await this.getCellValuesByRecord(id);
    const fields: { [key: string]: unknown } = {};
    for (const cell of cells) {
      fields[cell.field_id] = cell.value ? JSON.parse(cell.value as string) : null;
    }

    return { ...record, fields };
  }

  async getRecordsByTable(tableId: string, options?: RecordQueryOptions): Promise<{ records: RecordWithFields[]; next_cursor?: string; has_more: boolean }> {
    const limit = options?.limit || 50;
    const cursorPosition = options?.cursor ? parseInt(options.cursor, 10) : -1;

    // Get records
    const stmt = this.db.prepare(`
      SELECT * FROM records
      WHERE table_id = ? AND position > ?
      ORDER BY position
      LIMIT ?
    `);
    const rows = stmt.all(tableId, cursorPosition, limit + 1) as DbRow[];

    const hasMore = rows.length > limit;
    const recordRows = hasMore ? rows.slice(0, limit) : rows;

    // Get cell values for all records
    const records: RecordWithFields[] = [];
    for (const row of recordRows) {
      const record = this.rowToRecord(row);
      const cells = await this.getCellValuesByRecord(record.id);
      const fields: { [key: string]: unknown } = {};
      for (const cell of cells) {
        fields[cell.field_id] = cell.value ? JSON.parse(cell.value as string) : null;
      }
      records.push({ ...record, fields });
    }

    const nextCursor = hasMore && records.length > 0
      ? String(records[records.length - 1].position)
      : undefined;

    return { records, next_cursor: nextCursor, has_more: hasMore };
  }

  async updateRecord(id: string, updated_by: string): Promise<Record | null> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare('UPDATE records SET updated_by = ?, updated_at = ? WHERE id = ?');
    stmt.run(updated_by, now, id);
    return this.getRecord(id);
  }

  async deleteRecord(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM records WHERE id = ?');
    stmt.run(id);
  }

  async deleteRecordsByTable(tableId: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM records WHERE table_id = ?');
    stmt.run(tableId);
  }

  async getMaxRecordPosition(tableId: string): Promise<number> {
    const stmt = this.db.prepare('SELECT COALESCE(MAX(position), -1) as max_pos FROM records WHERE table_id = ?');
    const row = stmt.get(tableId) as DbRow;
    return row.max_pos as number;
  }

  async getRecordCount(tableId: string): Promise<number> {
    const stmt = this.db.prepare('SELECT COUNT(*) as count FROM records WHERE table_id = ?');
    const row = stmt.get(tableId) as DbRow;
    return row.count as number;
  }

  private rowToRecord(row: DbRow): Record {
    return {
      id: row.id as string,
      table_id: row.table_id as string,
      position: row.position as number,
      created_by: row.created_by as string,
      created_at: row.created_at as string,
      updated_by: row.updated_by as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Cell Values
  // ============================================================================

  async setCellValue(recordId: string, fieldId: string, value: unknown): Promise<CellValue> {
    const now = new Date().toISOString();
    const id = ulid();
    const jsonValue = JSON.stringify(value);

    // Extract searchable values
    let textValue: string | null = null;
    let numberValue: number | null = null;
    let dateValue: string | null = null;

    if (typeof value === 'string') {
      textValue = value;
    } else if (typeof value === 'number') {
      numberValue = value;
    } else if (value instanceof Date) {
      dateValue = value.toISOString();
    }

    const stmt = this.db.prepare(`
      INSERT INTO cell_values (id, record_id, field_id, value, text_value, number_value, date_value, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(record_id, field_id) DO UPDATE SET
        value = excluded.value,
        text_value = excluded.text_value,
        number_value = excluded.number_value,
        date_value = excluded.date_value,
        updated_at = excluded.updated_at
    `);
    stmt.run(id, recordId, fieldId, jsonValue, textValue, numberValue, dateValue, now);

    return {
      id,
      record_id: recordId,
      field_id: fieldId,
      value,
      text_value: textValue,
      number_value: numberValue,
      date_value: dateValue,
      updated_at: now,
    };
  }

  async getCellValue(recordId: string, fieldId: string): Promise<CellValue | null> {
    const stmt = this.db.prepare('SELECT * FROM cell_values WHERE record_id = ? AND field_id = ?');
    const row = stmt.get(recordId, fieldId) as DbRow | undefined;
    return row ? this.rowToCellValue(row) : null;
  }

  async getCellValuesByRecord(recordId: string): Promise<CellValue[]> {
    const stmt = this.db.prepare('SELECT * FROM cell_values WHERE record_id = ?');
    const rows = stmt.all(recordId) as DbRow[];
    return rows.map(row => this.rowToCellValue(row));
  }

  async deleteCellValue(recordId: string, fieldId: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM cell_values WHERE record_id = ? AND field_id = ?');
    stmt.run(recordId, fieldId);
  }

  async deleteCellValuesByRecord(recordId: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM cell_values WHERE record_id = ?');
    stmt.run(recordId);
  }

  async deleteCellValuesByField(fieldId: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM cell_values WHERE field_id = ?');
    stmt.run(fieldId);
  }

  private rowToCellValue(row: DbRow): CellValue {
    return {
      id: row.id as string,
      record_id: row.record_id as string,
      field_id: row.field_id as string,
      value: row.value ? JSON.parse(row.value as string) : null,
      text_value: row.text_value as string | null,
      number_value: row.number_value as number | null,
      date_value: row.date_value as string | null,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Views
  // ============================================================================

  async createView(input: CreateViewInput & { id: string; table_id: string; created_by: string; position: number }): Promise<View> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO views (id, table_id, name, type, filters, sorts, groups, field_config, settings, position, created_by, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      input.id,
      input.table_id,
      input.name,
      input.type,
      JSON.stringify(input.filters || []),
      JSON.stringify(input.sorts || []),
      JSON.stringify(input.groups || []),
      JSON.stringify(input.field_config || []),
      JSON.stringify(input.settings || {}),
      input.position,
      input.created_by,
      now,
      now
    );

    return {
      id: input.id,
      table_id: input.table_id,
      name: input.name,
      type: input.type,
      filters: input.filters || [],
      sorts: input.sorts || [],
      groups: input.groups || [],
      field_config: input.field_config || [],
      settings: input.settings || {},
      position: input.position,
      is_default: false,
      is_locked: false,
      created_by: input.created_by,
      created_at: now,
      updated_at: now,
    };
  }

  async getView(id: string): Promise<View | null> {
    const stmt = this.db.prepare('SELECT * FROM views WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToView(row) : null;
  }

  async getViewsByTable(tableId: string): Promise<View[]> {
    const stmt = this.db.prepare('SELECT * FROM views WHERE table_id = ? ORDER BY position');
    const rows = stmt.all(tableId) as DbRow[];
    return rows.map(row => this.rowToView(row));
  }

  async updateView(id: string, data: UpdateViewInput): Promise<View | null> {
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.name !== undefined) {
      updates.push('name = ?');
      values.push(data.name);
    }
    if (data.filters !== undefined) {
      updates.push('filters = ?');
      values.push(JSON.stringify(data.filters));
    }
    if (data.sorts !== undefined) {
      updates.push('sorts = ?');
      values.push(JSON.stringify(data.sorts));
    }
    if (data.groups !== undefined) {
      updates.push('groups = ?');
      values.push(JSON.stringify(data.groups));
    }
    if (data.field_config !== undefined) {
      updates.push('field_config = ?');
      values.push(JSON.stringify(data.field_config));
    }
    if (data.settings !== undefined) {
      updates.push('settings = ?');
      values.push(JSON.stringify(data.settings));
    }
    if (data.is_locked !== undefined) {
      updates.push('is_locked = ?');
      values.push(data.is_locked ? 1 : 0);
    }

    if (updates.length === 0) return this.getView(id);

    updates.push('updated_at = ?');
    values.push(new Date().toISOString());
    values.push(id);

    const stmt = this.db.prepare(`UPDATE views SET ${updates.join(', ')} WHERE id = ?`);
    stmt.run(...values);

    return this.getView(id);
  }

  async deleteView(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM views WHERE id = ?');
    stmt.run(id);
  }

  async reorderViews(tableId: string, viewIds: string[]): Promise<void> {
    const stmt = this.db.prepare('UPDATE views SET position = ? WHERE id = ? AND table_id = ?');
    viewIds.forEach((viewId, index) => {
      stmt.run(index, viewId, tableId);
    });
  }

  async getMaxViewPosition(tableId: string): Promise<number> {
    const stmt = this.db.prepare('SELECT COALESCE(MAX(position), -1) as max_pos FROM views WHERE table_id = ?');
    const row = stmt.get(tableId) as DbRow;
    return row.max_pos as number;
  }

  private rowToView(row: DbRow): View {
    return {
      id: row.id as string,
      table_id: row.table_id as string,
      name: row.name as string,
      type: row.type as View['type'],
      filters: row.filters ? JSON.parse(row.filters as string) : [],
      sorts: row.sorts ? JSON.parse(row.sorts as string) : [],
      groups: row.groups ? JSON.parse(row.groups as string) : [],
      field_config: row.field_config ? JSON.parse(row.field_config as string) : [],
      settings: row.settings ? JSON.parse(row.settings as string) : {},
      position: row.position as number,
      is_default: Boolean(row.is_default),
      is_locked: Boolean(row.is_locked),
      created_by: row.created_by as string,
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Comments
  // ============================================================================

  async createComment(input: CreateCommentInput & { id: string; record_id: string; author_id: string }): Promise<Comment> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO comments (id, record_id, parent_id, author_id, content, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      input.id,
      input.record_id,
      input.parent_id || null,
      input.author_id,
      JSON.stringify(input.content),
      now,
      now
    );

    return {
      id: input.id,
      record_id: input.record_id,
      parent_id: input.parent_id || null,
      author_id: input.author_id,
      content: input.content,
      is_resolved: false,
      created_at: now,
      updated_at: now,
    };
  }

  async getComment(id: string): Promise<Comment | null> {
    const stmt = this.db.prepare('SELECT * FROM comments WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToComment(row) : null;
  }

  async getCommentsByRecord(recordId: string): Promise<Comment[]> {
    const stmt = this.db.prepare('SELECT * FROM comments WHERE record_id = ? ORDER BY created_at');
    const rows = stmt.all(recordId) as DbRow[];
    return rows.map(row => this.rowToComment(row));
  }

  async updateComment(id: string, data: UpdateCommentInput): Promise<Comment | null> {
    const updates: string[] = [];
    const values: unknown[] = [];

    if (data.content !== undefined) {
      updates.push('content = ?');
      values.push(JSON.stringify(data.content));
    }
    if (data.is_resolved !== undefined) {
      updates.push('is_resolved = ?');
      values.push(data.is_resolved ? 1 : 0);
    }

    if (updates.length === 0) return this.getComment(id);

    updates.push('updated_at = ?');
    values.push(new Date().toISOString());
    values.push(id);

    const stmt = this.db.prepare(`UPDATE comments SET ${updates.join(', ')} WHERE id = ?`);
    stmt.run(...values);

    return this.getComment(id);
  }

  async deleteComment(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM comments WHERE id = ?');
    stmt.run(id);
  }

  private rowToComment(row: DbRow): Comment {
    return {
      id: row.id as string,
      record_id: row.record_id as string,
      parent_id: row.parent_id as string | null,
      author_id: row.author_id as string,
      content: row.content ? JSON.parse(row.content as string) : {},
      is_resolved: Boolean(row.is_resolved),
      created_at: row.created_at as string,
      updated_at: row.updated_at as string,
    };
  }

  // ============================================================================
  // Shares
  // ============================================================================

  async createShare(input: CreateShareInput & { id: string; base_id: string; created_by: string; token?: string }): Promise<Share> {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO shares (id, base_id, type, permission, email, token, password, expires_at, created_by, created_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      input.id,
      input.base_id,
      input.type,
      input.permission,
      input.email || null,
      input.token || null,
      input.password || null,
      input.expires_at || null,
      input.created_by,
      now
    );

    return {
      id: input.id,
      base_id: input.base_id,
      type: input.type,
      permission: input.permission,
      user_id: null,
      email: input.email || null,
      token: input.token || null,
      password: input.password || null,
      expires_at: input.expires_at || null,
      created_by: input.created_by,
      created_at: now,
    };
  }

  async getShare(id: string): Promise<Share | null> {
    const stmt = this.db.prepare('SELECT * FROM shares WHERE id = ?');
    const row = stmt.get(id) as DbRow | undefined;
    return row ? this.rowToShare(row) : null;
  }

  async getShareByToken(token: string): Promise<Share | null> {
    const stmt = this.db.prepare('SELECT * FROM shares WHERE token = ?');
    const row = stmt.get(token) as DbRow | undefined;
    return row ? this.rowToShare(row) : null;
  }

  async getSharesByBase(baseId: string): Promise<Share[]> {
    const stmt = this.db.prepare('SELECT * FROM shares WHERE base_id = ? ORDER BY created_at DESC');
    const rows = stmt.all(baseId) as DbRow[];
    return rows.map(row => this.rowToShare(row));
  }

  async deleteShare(id: string): Promise<void> {
    const stmt = this.db.prepare('DELETE FROM shares WHERE id = ?');
    stmt.run(id);
  }

  private rowToShare(row: DbRow): Share {
    return {
      id: row.id as string,
      base_id: row.base_id as string,
      type: row.type as string,
      permission: row.permission as string,
      user_id: row.user_id as string | null,
      email: row.email as string | null,
      token: row.token as string | null,
      password: row.password as string | null,
      expires_at: row.expires_at as string | null,
      created_by: row.created_by as string,
      created_at: row.created_at as string,
    };
  }
}

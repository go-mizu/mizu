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
} from '../models';

// User Store
export interface UserStore {
  create(user: Omit<User, 'createdAt' | 'updatedAt'>): Promise<User>;
  getById(id: string): Promise<User | null>;
  getByEmail(email: string): Promise<User | null>;
  update(id: string, data: Partial<User>): Promise<User>;
  delete(id: string): Promise<void>;
}

// Session Store
export interface SessionStore {
  create(session: Omit<Session, 'createdAt'>): Promise<Session>;
  getById(id: string): Promise<Session | null>;
  deleteById(id: string): Promise<void>;
  deleteByUserId(userId: string): Promise<void>;
  deleteExpired(): Promise<void>;
}

// Workspace Store
export interface WorkspaceStore {
  create(workspace: Omit<Workspace, 'createdAt' | 'updatedAt'>): Promise<Workspace>;
  getById(id: string): Promise<Workspace | null>;
  getBySlug(slug: string): Promise<Workspace | null>;
  listByUser(userId: string): Promise<Workspace[]>;
  update(id: string, data: Partial<Workspace>): Promise<Workspace>;
  delete(id: string): Promise<void>;
}

// Member Store
export interface MemberStore {
  create(member: Omit<Member, 'createdAt'>): Promise<Member>;
  getById(id: string): Promise<Member | null>;
  getByWorkspaceAndUser(workspaceId: string, userId: string): Promise<Member | null>;
  listByWorkspace(workspaceId: string): Promise<Member[]>;
  listByUser(userId: string): Promise<Member[]>;
  update(id: string, data: Partial<Member>): Promise<Member>;
  delete(id: string): Promise<void>;
}

// Page Store
export interface PageStore {
  create(page: Omit<Page, 'createdAt' | 'updatedAt'>): Promise<Page>;
  getById(id: string): Promise<Page | null>;
  listByWorkspace(workspaceId: string, options?: {
    parentId?: string | null;
    includeArchived?: boolean;
  }): Promise<Page[]>;
  listByParent(parentId: string, parentType: string): Promise<Page[]>;
  listByDatabase(databaseId: string, options?: {
    cursor?: string;
    limit?: number;
    filter?: FilterGroup;
    sorts?: Sort[];
  }): Promise<PaginatedResult<Page>>;
  update(id: string, data: Partial<Page>): Promise<Page>;
  delete(id: string): Promise<void>;
  getHierarchy(id: string): Promise<{ id: string; title: string; icon?: string | null }[]>;
}

// Block Store
export interface BlockStore {
  create(block: Omit<Block, 'createdAt' | 'updatedAt'>): Promise<Block>;
  getById(id: string): Promise<Block | null>;
  listByPage(pageId: string): Promise<Block[]>;
  listByParent(parentId: string): Promise<Block[]>;
  update(id: string, data: Partial<Block>): Promise<Block>;
  delete(id: string): Promise<void>;
  deleteByPage(pageId: string): Promise<void>;
  getMaxPosition(pageId: string, parentId?: string): Promise<number>;
  batchUpsert(pageId: string, blocks: Block[]): Promise<void>;
}

// Database Store
export interface DatabaseStore {
  create(database: Omit<Database, 'createdAt' | 'updatedAt'>): Promise<Database>;
  getById(id: string): Promise<Database | null>;
  getByPageId(pageId: string): Promise<Database | null>;
  listByWorkspace(workspaceId: string): Promise<Database[]>;
  update(id: string, data: Partial<Database>): Promise<Database>;
  delete(id: string): Promise<void>;
}

// View Store
export interface ViewStore {
  create(view: Omit<View, 'createdAt' | 'updatedAt'>): Promise<View>;
  getById(id: string): Promise<View | null>;
  listByDatabase(databaseId: string): Promise<View[]>;
  update(id: string, data: Partial<View>): Promise<View>;
  delete(id: string): Promise<void>;
  getMaxPosition(databaseId: string): Promise<number>;
}

// Comment Store
export interface CommentStore {
  create(comment: Omit<Comment, 'createdAt' | 'updatedAt'>): Promise<Comment>;
  getById(id: string): Promise<Comment | null>;
  listByTarget(targetType: string, targetId: string): Promise<Comment[]>;
  listByPage(pageId: string): Promise<Comment[]>;
  update(id: string, data: Partial<Comment>): Promise<Comment>;
  delete(id: string): Promise<void>;
}

// Share Store
export interface ShareStore {
  create(share: Omit<Share, 'createdAt' | 'updatedAt'>): Promise<Share>;
  getById(id: string): Promise<Share | null>;
  getByToken(token: string): Promise<Share | null>;
  listByPage(pageId: string): Promise<Share[]>;
  update(id: string, data: Partial<Share>): Promise<Share>;
  delete(id: string): Promise<void>;
}

// Favorite Store
export interface FavoriteStore {
  create(favorite: Omit<Favorite, 'createdAt'>): Promise<Favorite>;
  getById(id: string): Promise<Favorite | null>;
  getByUserAndPage(userId: string, pageId: string): Promise<Favorite | null>;
  listByUser(userId: string, workspaceId: string): Promise<Favorite[]>;
  delete(id: string): Promise<void>;
  deleteByUserAndPage(userId: string, pageId: string): Promise<void>;
  getMaxPosition(userId: string): Promise<number>;
}

// Activity Store
export interface ActivityStore {
  create(activity: Omit<Activity, 'createdAt'>): Promise<Activity>;
  listByWorkspace(workspaceId: string, options?: {
    cursor?: string;
    limit?: number;
  }): Promise<PaginatedResult<Activity>>;
  listByUser(userId: string, options?: {
    cursor?: string;
    limit?: number;
  }): Promise<PaginatedResult<Activity>>;
}

// Notification Store
export interface NotificationStore {
  create(notification: Omit<Notification, 'createdAt'>): Promise<Notification>;
  getById(id: string): Promise<Notification | null>;
  listByUser(userId: string, options?: {
    unreadOnly?: boolean;
    cursor?: string;
    limit?: number;
  }): Promise<PaginatedResult<Notification>>;
  markAsRead(id: string): Promise<void>;
  markAllAsRead(userId: string): Promise<void>;
  delete(id: string): Promise<void>;
}

// Synced Block Store
export interface SyncedBlockStore {
  create(syncedBlock: Omit<SyncedBlock, 'createdAt' | 'updatedAt'>): Promise<SyncedBlock>;
  getById(id: string): Promise<SyncedBlock | null>;
  listByWorkspace(workspaceId: string): Promise<SyncedBlock[]>;
  listByPage(pageId: string): Promise<SyncedBlock[]>;
  update(id: string, data: Partial<SyncedBlock>): Promise<SyncedBlock>;
  delete(id: string): Promise<void>;
}

// Main Store Interface
export interface Store {
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

  // Lifecycle
  init(): Promise<void>;
  close(): Promise<void>;
}

// Store Driver Type
export type StoreDriver = 'd1' | 'sqlite' | 'postgres';

// Store Configuration
export interface StoreConfig {
  driver: StoreDriver;
  // D1 (Cloudflare)
  d1?: D1Database;
  // SQLite
  sqlitePath?: string;
  // PostgreSQL
  postgresUrl?: string;
  postgresSchema?: string; // Optional schema for test isolation
}

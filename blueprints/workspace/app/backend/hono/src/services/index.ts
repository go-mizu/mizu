import type { Store } from '../store/types';
import { UserService } from './users';
import { WorkspaceService } from './workspaces';
import { PageService } from './pages';
import { BlockService } from './blocks';
import { DatabaseService } from './databases';
import { ViewService } from './views';
import { RowService } from './rows';
import { CommentService } from './comments';
import { ShareService } from './sharing';
import { FavoriteService } from './favorites';
import { SearchService } from './search';
import { SyncedBlockService } from './synced-blocks';

export interface Services {
  users: UserService;
  workspaces: WorkspaceService;
  pages: PageService;
  blocks: BlockService;
  databases: DatabaseService;
  views: ViewService;
  rows: RowService;
  comments: CommentService;
  sharing: ShareService;
  favorites: FavoriteService;
  search: SearchService;
  syncedBlocks: SyncedBlockService;
}

export function createServices(store: Store): Services {
  return {
    users: new UserService(store),
    workspaces: new WorkspaceService(store),
    pages: new PageService(store),
    blocks: new BlockService(store),
    databases: new DatabaseService(store),
    views: new ViewService(store),
    rows: new RowService(store),
    comments: new CommentService(store),
    sharing: new ShareService(store),
    favorites: new FavoriteService(store),
    search: new SearchService(store),
    syncedBlocks: new SyncedBlockService(store),
  };
}

export * from './users';
export * from './workspaces';
export * from './pages';
export * from './blocks';
export * from './databases';
export * from './views';
export * from './rows';
export * from './comments';
export * from './sharing';
export * from './favorites';
export * from './search';
export * from './synced-blocks';

import { Hono } from 'hono';
import type { Env, Variables } from '../env';
import { authRoutes } from './auth';
import { workspaceRoutes } from './workspaces';
import { pageRoutes } from './pages';
import { blockRoutes } from './blocks';
import { databaseRoutes } from './databases';
import { viewRoutes } from './views';
import { rowRoutes } from './rows';
import { commentRoutes } from './comments';
import { shareRoutes } from './sharing';
import { favoriteRoutes } from './favorites';
import { searchRoutes } from './search';
import { syncedBlockRoutes } from './synced-blocks';
import { uiRoutes } from './ui';

export function createRoutes() {
  const app = new Hono<{ Bindings: Env; Variables: Variables }>();

  // Health check
  app.get('/api/v1/health', (c) => {
    return c.json({ status: 'ok', timestamp: new Date().toISOString() });
  });

  // API routes
  app.route('/api/v1/auth', authRoutes);
  app.route('/api/v1/workspaces', workspaceRoutes);
  app.route('/api/v1/pages', pageRoutes);
  app.route('/api/v1/blocks', blockRoutes);
  app.route('/api/v1/databases', databaseRoutes);
  app.route('/api/v1/views', viewRoutes);
  app.route('/api/v1/rows', rowRoutes);
  app.route('/api/v1/comments', commentRoutes);
  app.route('/api/v1/shares', shareRoutes);
  app.route('/api/v1/favorites', favoriteRoutes);
  app.route('/api/v1/synced-blocks', syncedBlockRoutes);

  // Search routes (under workspaces)
  app.route('/api/v1', searchRoutes);

  // UI routes
  app.route('/', uiRoutes);

  return app;
}

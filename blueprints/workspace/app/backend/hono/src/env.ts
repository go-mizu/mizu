export interface Env {
  // Cloudflare bindings
  DB: D1Database;
  UPLOADS: R2Bucket;

  // Environment variables
  ENVIRONMENT: string;
  DEV_MODE?: string;
  SESSION_SECRET?: string;
}

export interface Variables {
  user: User | null;
  userId: string | null;
  store: Store;
}

import type { User } from './models/user';
import type { Store } from './store/types';

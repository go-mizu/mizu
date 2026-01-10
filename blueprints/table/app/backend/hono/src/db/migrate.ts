/**
 * Database migration script
 *
 * Run: pnpm db:migrate
 */

import { existsSync, mkdirSync } from 'fs';
import { dirname } from 'path';
import { SqliteDriver } from './sqlite.js';

const dbPath = process.env.DATABASE_PATH || './data/table.db';

// Ensure data directory exists
const dbDir = dirname(dbPath);
if (!existsSync(dbDir)) {
  mkdirSync(dbDir, { recursive: true });
}

console.log(`Running migrations for: ${dbPath}`);

const db = new SqliteDriver(dbPath);

try {
  await db.ensure();
  console.log('Migrations completed successfully');
} catch (error) {
  console.error('Migration failed:', error);
  process.exit(1);
} finally {
  await db.close();
}

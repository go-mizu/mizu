/**
 * Database migration script
 *
 * Usage:
 *   # SQLite (local file)
 *   DATABASE_PATH=./data/spreadsheet.db tsx src/db/migrate.ts
 *
 *   # PostgreSQL
 *   DATABASE_URL=postgres://... tsx src/db/migrate.ts
 *
 *   # Cloudflare D1 (use wrangler)
 *   wrangler d1 execute spreadsheet-db --file=src/db/schema.sql
 */

import { schema, postgresSchema } from './schema.js';

async function main() {
  const databaseUrl = process.env.DATABASE_URL;
  const databasePath = process.env.DATABASE_PATH;

  if (databaseUrl) {
    // PostgreSQL migration
    console.log('Running PostgreSQL migrations...');
    const postgres = await import('postgres');
    const sql = postgres.default(databaseUrl);

    try {
      await sql.unsafe(postgresSchema);
      console.log('PostgreSQL migrations completed successfully!');
    } catch (error) {
      console.error('Migration failed:', error);
      process.exit(1);
    } finally {
      await sql.end();
    }
  } else if (databasePath) {
    // SQLite migration
    console.log(`Running SQLite migrations on ${databasePath}...`);
    const BetterSqlite3 = await import('better-sqlite3');
    const db = new BetterSqlite3.default(databasePath);

    try {
      // Execute each statement separately
      const statements = schema
        .split(';')
        .map(s => s.trim())
        .filter(s => s.length > 0 && !s.startsWith('--'));

      for (const stmt of statements) {
        db.exec(stmt);
      }
      console.log('SQLite migrations completed successfully!');
    } catch (error) {
      console.error('Migration failed:', error);
      process.exit(1);
    } finally {
      db.close();
    }
  } else {
    console.log(`
Database Migration Script

Usage:
  # SQLite (local file)
  DATABASE_PATH=./data/spreadsheet.db npm run db:migrate

  # PostgreSQL
  DATABASE_URL=postgres://user:pass@host:5432/db npm run db:migrate

  # Cloudflare D1
  wrangler d1 execute spreadsheet-db --file=src/db/schema.sql
`);
    process.exit(1);
  }
}

main();

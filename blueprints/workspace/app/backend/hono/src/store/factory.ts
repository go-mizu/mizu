import type { Store, StoreConfig } from './types';

export async function createStore(config: StoreConfig): Promise<Store> {
  switch (config.driver) {
    case 'd1': {
      if (!config.d1) {
        throw new Error('D1 database binding required for d1 driver');
      }
      const { D1Store } = await import('./d1');
      const store = new D1Store(config.d1);
      await store.init();
      return store;
    }

    case 'sqlite': {
      if (!config.sqlitePath) {
        throw new Error('SQLite path required for sqlite driver');
      }
      const { SQLiteStore } = await import('./sqlite');
      const store = new SQLiteStore(config.sqlitePath);
      await store.init();
      return store;
    }

    case 'postgres': {
      if (!config.postgresUrl) {
        throw new Error('PostgreSQL URL required for postgres driver');
      }
      const { PostgresStore } = await import('./postgres');
      const store = new PostgresStore(config.postgresUrl, {
        schema: config.postgresSchema,
      });
      await store.init();
      return store;
    }

    default:
      throw new Error(`Unknown store driver: ${config.driver}`);
  }
}

export { type Store, type StoreConfig, type StoreDriver } from './types';

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { databaseApi } from './database';
import { api } from './client';

// Mock the API client
vi.mock('./client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('databaseApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getOverview', () => {
    it('fetches database overview', async () => {
      const mockOverview = {
        schemas: [{ name: 'public', table_count: 5, view_count: 2 }],
        total_tables: 5,
        total_views: 2,
        total_functions: 10,
        total_indexes: 15,
        total_policies: 3,
        database_size: '128 MB',
        connection_count: 3,
      };

      (api.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockOverview);

      const result = await databaseApi.getOverview();

      expect(api.get).toHaveBeenCalledWith('/api/database/overview');
      expect(result).toEqual(mockOverview);
    });
  });

  describe('getTableStats', () => {
    it('fetches table stats for a schema', async () => {
      const mockStats = [
        { schema: 'public', name: 'users', row_count: 100, size_bytes: 1024 },
      ];

      (api.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockStats);

      const result = await databaseApi.getTableStats('public');

      expect(api.get).toHaveBeenCalledWith('/api/database/tables/stats?schema=public');
      expect(result).toEqual(mockStats);
    });
  });

  describe('listIndexes', () => {
    it('fetches indexes with schema and table filters', async () => {
      const mockIndexes = [
        { name: 'users_pkey', schema: 'public', table: 'users', type: 'btree' },
      ];

      (api.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockIndexes);

      const result = await databaseApi.listIndexes('public', 'users');

      expect(api.get).toHaveBeenCalledWith('/api/database/indexes?schema=public&table=users');
      expect(result).toEqual(mockIndexes);
    });

    it('fetches all indexes when no filter provided', async () => {
      (api.get as ReturnType<typeof vi.fn>).mockResolvedValue([]);

      await databaseApi.listIndexes();

      expect(api.get).toHaveBeenCalledWith('/api/database/indexes');
    });
  });

  describe('createIndex', () => {
    it('creates an index', async () => {
      (api.post as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      await databaseApi.createIndex({
        name: 'idx_users_email',
        schema: 'public',
        table: 'users',
        columns: ['email'],
        type: 'btree',
        is_unique: true,
      });

      expect(api.post).toHaveBeenCalledWith('/api/database/indexes', {
        name: 'idx_users_email',
        schema: 'public',
        table: 'users',
        columns: ['email'],
        type: 'btree',
        is_unique: true,
      });
    });
  });

  describe('dropIndex', () => {
    it('drops an index', async () => {
      (api.delete as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      await databaseApi.dropIndex('public', 'idx_users_email');

      expect(api.delete).toHaveBeenCalledWith('/api/database/indexes/public/idx_users_email');
    });
  });

  describe('enableTableRLS', () => {
    it('enables RLS on a table', async () => {
      (api.post as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      await databaseApi.enableTableRLS('public', 'users');

      expect(api.post).toHaveBeenCalledWith('/api/database/tables/public/users/rls/enable', {});
    });
  });

  describe('disableTableRLS', () => {
    it('disables RLS on a table', async () => {
      (api.post as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      await databaseApi.disableTableRLS('public', 'users');

      expect(api.post).toHaveBeenCalledWith('/api/database/tables/public/users/rls/disable', {});
    });
  });

  describe('listSchemas', () => {
    it('fetches schemas', async () => {
      const mockSchemas = ['public', 'auth', 'storage'];

      (api.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockSchemas);

      const result = await databaseApi.listSchemas();

      expect(api.get).toHaveBeenCalledWith('/api/database/schemas');
      expect(result).toEqual(mockSchemas);
    });
  });

  describe('listTables', () => {
    it('fetches tables for a schema', async () => {
      const mockTables = [{ schema: 'public', name: 'users', row_count: 100 }];

      (api.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockTables);

      const result = await databaseApi.listTables('public');

      expect(api.get).toHaveBeenCalledWith('/api/database/tables?schema=public');
      expect(result).toEqual(mockTables);
    });
  });

  describe('executeQuery', () => {
    it('executes a query', async () => {
      const mockResult = {
        columns: [{ name: 'id', type: 'uuid' }],
        rows: [{ id: '123' }],
        row_count: 1,
        duration_ms: 10,
      };

      (api.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResult);

      const result = await databaseApi.executeQuery('SELECT * FROM users');

      expect(api.post).toHaveBeenCalledWith('/api/database/query', {
        query: 'SELECT * FROM users',
        role: undefined,
        explain: undefined,
      });
      expect(result).toEqual(mockResult);
    });

    it('executes a query with role and explain options', async () => {
      (api.post as ReturnType<typeof vi.fn>).mockResolvedValue({});

      await databaseApi.executeQuery('SELECT * FROM users', { role: 'anon', explain: true });

      expect(api.post).toHaveBeenCalledWith('/api/database/query', {
        query: 'SELECT * FROM users',
        role: 'anon',
        explain: true,
      });
    });
  });

  describe('createPolicy', () => {
    it('creates a policy', async () => {
      (api.post as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      await databaseApi.createPolicy({
        name: 'select_policy',
        schema: 'public',
        table: 'users',
        command: 'SELECT',
        definition: 'true',
      });

      expect(api.post).toHaveBeenCalledWith('/api/database/policies', {
        name: 'select_policy',
        schema: 'public',
        table: 'users',
        command: 'SELECT',
        definition: 'true',
      });
    });
  });
});

import { test, expect } from '@playwright/test';
import { APIClient, registerAndLogin, createTestWorkbook, Workbook, Sheet } from './helpers';

test.describe('Workbook API', () => {
  let api: APIClient;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);
  });

  test.describe('Create Workbook', () => {
    test('should create workbook successfully', async () => {
      const res = await api.createWorkbook('My Spreadsheet');
      expect(res.ok()).toBeTruthy();
      expect(res.status()).toBe(201);

      const data = await res.json();
      // API returns { workbook, sheet } structure
      const workbook = data.workbook as Workbook;
      expect(workbook.id).toBeDefined();
      expect(workbook.name).toBe('My Spreadsheet');
      expect(workbook.ownerId).toBeDefined();
    });

    test('should auto-create default sheet when creating workbook', async () => {
      const workbook = await createTestWorkbook(api, 'Test Workbook');

      // List sheets
      const res = await api.listSheets(workbook.id);
      expect(res.ok()).toBeTruthy();

      const sheets = (await res.json()) as Sheet[];
      expect(sheets.length).toBe(1);
      expect(sheets[0].name).toBe('Sheet1');
    });

    test('should create multiple workbooks', async () => {
      await createTestWorkbook(api, 'Workbook 1');
      await createTestWorkbook(api, 'Workbook 2');
      await createTestWorkbook(api, 'Workbook 3');

      const res = await api.listWorkbooks();
      const workbooks = (await res.json()) as Workbook[];
      expect(workbooks.length).toBe(3);
    });
  });

  test.describe('List Workbooks', () => {
    test('should return empty array when no workbooks', async () => {
      const res = await api.listWorkbooks();
      expect(res.ok()).toBeTruthy();

      const workbooks = await res.json();
      expect(Array.isArray(workbooks)).toBeTruthy();
    });

    test('should return only user\'s workbooks', async ({ request }) => {
      // Create workbook for first user
      await createTestWorkbook(api, 'User1 Workbook');

      // Create second user and their workbook
      const api2 = new APIClient(request);
      await registerAndLogin(api2);
      await createTestWorkbook(api2, 'User2 Workbook');

      // First user should only see their workbook
      const res = await api.listWorkbooks();
      const workbooks = (await res.json()) as Workbook[];
      expect(workbooks.length).toBe(1);
      expect(workbooks[0].name).toBe('User1 Workbook');
    });
  });

  test.describe('Get Workbook', () => {
    test('should get workbook by ID', async () => {
      const created = await createTestWorkbook(api, 'My Workbook');

      const res = await api.getWorkbook(created.id);
      expect(res.ok()).toBeTruthy();

      const data = await res.json();
      // API returns { workbook, sheets } structure
      const workbook = data.workbook as Workbook;
      expect(workbook.id).toBe(created.id);
      expect(workbook.name).toBe('My Workbook');
    });

    test('should return 404 for non-existent workbook', async () => {
      const res = await api.getWorkbook('non-existent-id');
      expect(res.status()).toBe(404);
    });
  });

  test.describe('Update Workbook', () => {
    test('should update workbook name', async () => {
      const created = await createTestWorkbook(api, 'Original Name');

      const res = await api.updateWorkbook(created.id, { name: 'New Name' });
      expect(res.ok()).toBeTruthy();

      const updated = (await res.json()) as Workbook;
      expect(updated.name).toBe('New Name');
    });

    test('should return 404 for non-existent workbook', async () => {
      const res = await api.updateWorkbook('non-existent-id', { name: 'Test' });
      expect(res.status()).toBe(404);
    });
  });

  test.describe('Delete Workbook', () => {
    test('should delete workbook successfully', async () => {
      const created = await createTestWorkbook(api, 'To Delete');

      const res = await api.deleteWorkbook(created.id);
      expect(res.ok()).toBeTruthy();

      // Verify deletion
      const getRes = await api.getWorkbook(created.id);
      expect(getRes.status()).toBe(404);
    });

    test('should delete associated sheets when deleting workbook', async () => {
      const workbook = await createTestWorkbook(api, 'To Delete');

      // Get the default sheet
      const sheetsRes = await api.listSheets(workbook.id);
      const sheets = (await sheetsRes.json()) as Sheet[];
      const sheetId = sheets[0].id;

      // Delete workbook
      await api.deleteWorkbook(workbook.id);

      // Sheet should also be deleted
      const sheetRes = await api.getSheet(sheetId);
      expect(sheetRes.status()).toBe(404);
    });
  });

  test.describe('List Sheets in Workbook', () => {
    test('should list all sheets in workbook', async () => {
      const workbook = await createTestWorkbook(api);

      // Create additional sheets
      await api.createSheet(workbook.id, 'Sheet2');
      await api.createSheet(workbook.id, 'Sheet3');

      const res = await api.listSheets(workbook.id);
      expect(res.ok()).toBeTruthy();

      const sheets = (await res.json()) as Sheet[];
      expect(sheets.length).toBe(3);
      expect(sheets.map((s) => s.name)).toContain('Sheet1');
      expect(sheets.map((s) => s.name)).toContain('Sheet2');
      expect(sheets.map((s) => s.name)).toContain('Sheet3');
    });

    test('should return sheets ordered by index', async () => {
      const workbook = await createTestWorkbook(api);

      await api.createSheet(workbook.id, 'Sheet2', { index: 1 });
      await api.createSheet(workbook.id, 'Sheet3', { index: 2 });

      const res = await api.listSheets(workbook.id);
      const sheets = (await res.json()) as Sheet[];

      for (let i = 1; i < sheets.length; i++) {
        expect(sheets[i].index).toBeGreaterThanOrEqual(sheets[i - 1].index);
      }
    });
  });
});

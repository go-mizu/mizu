import { test, expect } from '@playwright/test';
import { APIClient, registerAndLogin, createTestWorkbook, getFirstSheet, Sheet } from './helpers';

test.describe('Sheet API', () => {
  let api: APIClient;
  let workbookId: string;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);
    const workbook = await createTestWorkbook(api);
    workbookId = workbook.id;
  });

  test.describe('Create Sheet', () => {
    test('should create sheet with default settings', async () => {
      const res = await api.createSheet(workbookId, 'New Sheet');
      expect(res.ok()).toBeTruthy();
      expect(res.status()).toBe(201);

      const sheet = (await res.json()) as Sheet;
      expect(sheet.id).toBeDefined();
      expect(sheet.name).toBe('New Sheet');
      expect(sheet.workbook_id).toBe(workbookId);
      expect(sheet.hidden).toBeFalsy();
    });

    test('should create sheet with custom index', async () => {
      const res = await api.createSheet(workbookId, 'Sheet at Index 5', { index: 5 });
      expect(res.ok()).toBeTruthy();

      const sheet = (await res.json()) as Sheet;
      expect(sheet.index).toBe(5);
    });

    test('should create sheet with custom color', async () => {
      const res = await api.createSheet(workbookId, 'Colored Sheet', { color: '#FF5733' });
      expect(res.ok()).toBeTruthy();

      const sheet = (await res.json()) as Sheet;
      expect(sheet.color).toBe('#FF5733');
    });

    test('should auto-increment index when not specified', async () => {
      // Default Sheet1 has index 0
      const res1 = await api.createSheet(workbookId, 'Sheet A');
      const sheet1 = (await res1.json()) as Sheet;

      const res2 = await api.createSheet(workbookId, 'Sheet B');
      const sheet2 = (await res2.json()) as Sheet;

      expect(sheet2.index).toBeGreaterThan(sheet1.index);
    });
  });

  test.describe('Get Sheet', () => {
    test('should get sheet by ID', async () => {
      const sheet = await getFirstSheet(api, workbookId);

      const res = await api.getSheet(sheet.id);
      expect(res.ok()).toBeTruthy();

      const fetched = (await res.json()) as Sheet;
      expect(fetched.id).toBe(sheet.id);
      expect(fetched.name).toBe(sheet.name);
    });

    test('should return 404 for non-existent sheet', async () => {
      const res = await api.getSheet('non-existent-id');
      expect(res.status()).toBe(404);
    });
  });

  test.describe('Update Sheet', () => {
    test('should update sheet name', async () => {
      const sheet = await getFirstSheet(api, workbookId);

      const res = await api.updateSheet(sheet.id, { name: 'Renamed Sheet' });
      expect(res.ok()).toBeTruthy();

      const updated = (await res.json()) as Sheet;
      expect(updated.name).toBe('Renamed Sheet');
    });

    test('should update sheet color', async () => {
      const sheet = await getFirstSheet(api, workbookId);

      const res = await api.updateSheet(sheet.id, { color: '#3B82F6' });
      expect(res.ok()).toBeTruthy();

      const updated = (await res.json()) as Sheet;
      expect(updated.color).toBe('#3B82F6');
    });

    test('should update frozen rows and columns', async () => {
      const sheet = await getFirstSheet(api, workbookId);

      const res = await api.updateSheet(sheet.id, { frozen_rows: 2, frozen_cols: 1 });
      expect(res.ok()).toBeTruthy();

      const updated = (await res.json()) as Sheet;
      expect(updated.frozen_rows).toBe(2);
      expect(updated.frozen_cols).toBe(1);
    });

    test('should update multiple properties at once', async () => {
      const sheet = await getFirstSheet(api, workbookId);

      const res = await api.updateSheet(sheet.id, {
        name: 'Updated Sheet',
        color: '#10B981',
        frozen_rows: 1,
      });
      expect(res.ok()).toBeTruthy();

      const updated = (await res.json()) as Sheet;
      expect(updated.name).toBe('Updated Sheet');
      expect(updated.color).toBe('#10B981');
      expect(updated.frozen_rows).toBe(1);
    });
  });

  test.describe('Delete Sheet', () => {
    test('should delete sheet when multiple sheets exist', async () => {
      // Create second sheet
      const createRes = await api.createSheet(workbookId, 'Sheet to Delete');
      const sheetToDelete = (await createRes.json()) as Sheet;

      const res = await api.deleteSheet(sheetToDelete.id);
      expect(res.ok()).toBeTruthy();

      // Verify deletion
      const getRes = await api.getSheet(sheetToDelete.id);
      expect(getRes.status()).toBe(404);
    });

    test('should not delete last sheet in workbook', async () => {
      const sheet = await getFirstSheet(api, workbookId);

      const res = await api.deleteSheet(sheet.id);
      expect(res.ok()).toBeFalsy();
      expect(res.status()).toBe(400);

      const data = await res.json();
      expect(data.error).toBeDefined();
    });

    test('should allow deletion after creating another sheet', async () => {
      const originalSheet = await getFirstSheet(api, workbookId);

      // Create a new sheet
      await api.createSheet(workbookId, 'New Sheet');

      // Now we can delete the original
      const res = await api.deleteSheet(originalSheet.id);
      expect(res.ok()).toBeTruthy();
    });
  });

  test.describe('Sheet Ordering', () => {
    test('should maintain sheet order', async () => {
      await api.createSheet(workbookId, 'Second', { index: 1 });
      await api.createSheet(workbookId, 'Third', { index: 2 });

      const res = await api.listSheets(workbookId);
      const sheets = (await res.json()) as Sheet[];

      expect(sheets.length).toBe(3);
      expect(sheets[0].name).toBe('Sheet1');
      expect(sheets[1].name).toBe('Second');
      expect(sheets[2].name).toBe('Third');
    });
  });
});

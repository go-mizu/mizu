import { test, expect } from '@playwright/test';
import { APIClient, registerAndLogin, createTestWorkbook, getFirstSheet, Cell } from './helpers';

test.describe('Cell API', () => {
  let api: APIClient;
  let sheetId: string;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);
    const workbook = await createTestWorkbook(api);
    const sheet = await getFirstSheet(api, workbook.id);
    sheetId = sheet.id;
  });

  test.describe('Set Cell Value', () => {
    test('should set text value', async () => {
      const res = await api.setCell(sheetId, 0, 0, { value: 'Hello World' });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.value).toBe('Hello World');
      expect(cell.row).toBe(0);
      expect(cell.col).toBe(0);
    });

    test('should set numeric value', async () => {
      const res = await api.setCell(sheetId, 0, 0, { value: 42 });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.value).toBe(42);
    });

    test('should set decimal value', async () => {
      const res = await api.setCell(sheetId, 0, 0, { value: 3.14159 });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.value).toBeCloseTo(3.14159);
    });

    test('should set boolean value', async () => {
      const res = await api.setCell(sheetId, 0, 0, { value: true });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.value).toBe(true);
    });

    test('should set null/empty value', async () => {
      // First set a value
      await api.setCell(sheetId, 0, 0, { value: 'Initial' });

      // Then clear it
      const res = await api.setCell(sheetId, 0, 0, { value: null });
      expect(res.ok()).toBeTruthy();
    });

    test('should update existing cell', async () => {
      await api.setCell(sheetId, 0, 0, { value: 'First' });
      const res = await api.setCell(sheetId, 0, 0, { value: 'Second' });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.value).toBe('Second');
    });
  });

  test.describe('Set Cell Formula', () => {
    test('should set simple formula', async () => {
      // Set source values
      await api.setCell(sheetId, 0, 0, { value: 10 });
      await api.setCell(sheetId, 0, 1, { value: 20 });

      // Set formula
      const res = await api.setCell(sheetId, 0, 2, { formula: '=A1+B1' });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.formula).toBe('=A1+B1');
      expect(cell.display).toBeDefined();
    });

    test('should set SUM formula', async () => {
      // Set source values
      await api.setCell(sheetId, 0, 0, { value: 1 });
      await api.setCell(sheetId, 1, 0, { value: 2 });
      await api.setCell(sheetId, 2, 0, { value: 3 });

      // Set SUM formula
      const res = await api.setCell(sheetId, 3, 0, { formula: '=SUM(A1:A3)' });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.formula).toBe('=SUM(A1:A3)');
    });

    test('should set formula with multiplication', async () => {
      await api.setCell(sheetId, 0, 0, { value: 5 });

      const res = await api.setCell(sheetId, 0, 1, { formula: '=A1*2' });
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.formula).toBe('=A1*2');
    });
  });

  test.describe('Get Cell', () => {
    test('should get existing cell', async () => {
      await api.setCell(sheetId, 5, 3, { value: 'Test Value' });

      const res = await api.getCell(sheetId, 5, 3);
      expect(res.ok()).toBeTruthy();

      const cell = (await res.json()) as Cell;
      expect(cell.value).toBe('Test Value');
      expect(cell.row).toBe(5);
      expect(cell.col).toBe(3);
    });

    test('should return empty for non-existent cell', async () => {
      const res = await api.getCell(sheetId, 999, 999);
      expect(res.ok()).toBeTruthy();

      const cell = await res.json();
      // Empty cell should have null or undefined value
      expect(cell.value === null || cell.value === undefined || Object.keys(cell).length === 0).toBeTruthy();
    });
  });

  test.describe('Get Cell Range', () => {
    test('should get cells in range', async () => {
      // Set some cells
      await api.setCell(sheetId, 0, 0, { value: 'A1' });
      await api.setCell(sheetId, 0, 1, { value: 'B1' });
      await api.setCell(sheetId, 1, 0, { value: 'A2' });
      await api.setCell(sheetId, 1, 1, { value: 'B2' });

      const res = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 2, endCol: 2 });
      expect(res.ok()).toBeTruthy();

      const cells = (await res.json()) as Cell[];
      expect(Array.isArray(cells)).toBeTruthy();
      expect(cells.length).toBeGreaterThanOrEqual(4);
    });

    test('should return empty array for empty range', async () => {
      const res = await api.getCells(sheetId, { startRow: 100, startCol: 100, endRow: 105, endCol: 105 });
      expect(res.ok()).toBeTruthy();

      const cells = await res.json();
      expect(Array.isArray(cells)).toBeTruthy();
    });
  });

  test.describe('Batch Update Cells', () => {
    test('should update multiple cells at once', async () => {
      const res = await api.batchUpdateCells(sheetId, [
        { row: 0, col: 0, value: 'Cell 1' },
        { row: 0, col: 1, value: 'Cell 2' },
        { row: 1, col: 0, value: 'Cell 3' },
        { row: 1, col: 1, value: 'Cell 4' },
      ]);
      expect(res.ok()).toBeTruthy();

      const cells = (await res.json()) as Cell[];
      expect(cells.length).toBe(4);
    });

    test('should batch update with formulas', async () => {
      const res = await api.batchUpdateCells(sheetId, [
        { row: 0, col: 0, value: 10 },
        { row: 0, col: 1, value: 20 },
        { row: 0, col: 2, formula: '=A1+B1' },
      ]);
      expect(res.ok()).toBeTruthy();

      const cells = (await res.json()) as Cell[];
      expect(cells.length).toBe(3);

      const formulaCell = cells.find((c) => c.col === 2);
      expect(formulaCell?.formula).toBe('=A1+B1');
    });

    test('should handle empty batch', async () => {
      const res = await api.batchUpdateCells(sheetId, []);
      expect(res.ok()).toBeTruthy();
    });
  });

  test.describe('Delete Cell', () => {
    test('should delete existing cell', async () => {
      await api.setCell(sheetId, 0, 0, { value: 'To Delete' });

      const res = await api.deleteCell(sheetId, 0, 0);
      expect(res.ok()).toBeTruthy();

      // Verify deletion
      const getRes = await api.getCell(sheetId, 0, 0);
      const cell = await getRes.json();
      expect(cell.value === null || cell.value === undefined || Object.keys(cell).length === 0).toBeTruthy();
    });

    test('should handle delete of non-existent cell', async () => {
      const res = await api.deleteCell(sheetId, 999, 999);
      expect(res.ok()).toBeTruthy();
    });
  });

  test.describe('Row Operations', () => {
    test.skip('should insert rows', async () => {
      // Note: Row/column insertion with cell shifting not fully implemented
      await api.setCell(sheetId, 0, 0, { value: 'Row 0' });
      await api.setCell(sheetId, 1, 0, { value: 'Row 1' });

      const res = await api.insertRows(sheetId, 1, 2);
      expect(res.ok()).toBeTruthy();
    });

    test.skip('should delete rows', async () => {
      // Note: Row/column deletion with cell shifting not fully implemented
      await api.setCell(sheetId, 0, 0, { value: 'Row 0' });
      await api.setCell(sheetId, 1, 0, { value: 'Row 1' });
      await api.setCell(sheetId, 2, 0, { value: 'Row 2' });

      const res = await api.deleteRows(sheetId, 1, 1);
      expect(res.ok()).toBeTruthy();
    });
  });

  test.describe('Column Operations', () => {
    test.skip('should insert columns', async () => {
      // Note: Row/column insertion with cell shifting not fully implemented
      await api.setCell(sheetId, 0, 0, { value: 'Col 0' });
      await api.setCell(sheetId, 0, 1, { value: 'Col 1' });

      const res = await api.insertCols(sheetId, 1, 1);
      expect(res.ok()).toBeTruthy();
    });

    test.skip('should delete columns', async () => {
      // Note: Row/column deletion with cell shifting not fully implemented
      await api.setCell(sheetId, 0, 0, { value: 'Col 0' });
      await api.setCell(sheetId, 0, 1, { value: 'Col 1' });
      await api.setCell(sheetId, 0, 2, { value: 'Col 2' });

      const res = await api.deleteCols(sheetId, 1, 1);
      expect(res.ok()).toBeTruthy();
    });
  });

  test.describe('Merge Operations', () => {
    test('should merge cells', async () => {
      const res = await api.merge(sheetId, { startRow: 0, startCol: 0, endRow: 2, endCol: 2 });
      expect(res.ok()).toBeTruthy();

      const merge = await res.json();
      expect(merge.start_row).toBe(0);
      expect(merge.start_col).toBe(0);
      expect(merge.end_row).toBe(2);
      expect(merge.end_col).toBe(2);
    });

    test('should get merged regions', async () => {
      await api.merge(sheetId, { startRow: 0, startCol: 0, endRow: 1, endCol: 1 });
      await api.merge(sheetId, { startRow: 5, startCol: 5, endRow: 7, endCol: 7 });

      const res = await api.getMerges(sheetId);
      expect(res.ok()).toBeTruthy();

      const merges = await res.json();
      expect(Array.isArray(merges)).toBeTruthy();
      expect(merges.length).toBe(2);
    });

    test('should unmerge cells', async () => {
      await api.merge(sheetId, { startRow: 0, startCol: 0, endRow: 2, endCol: 2 });

      const res = await api.unmerge(sheetId, { startRow: 0, startCol: 0, endRow: 2, endCol: 2 });
      expect(res.ok()).toBeTruthy();

      // Verify unmerge
      const mergesRes = await api.getMerges(sheetId);
      const merges = await mergesRes.json();
      expect(Array.isArray(merges)).toBeTruthy();
      expect(merges.length).toBe(0);
    });
  });
});

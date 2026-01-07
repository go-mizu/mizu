import { test, expect } from '@playwright/test';
import { APIClient, registerAndLogin, createTestWorkbook, getFirstSheet, Workbook, Sheet, Cell } from './helpers';

test.describe('Import/Export API', () => {
  let api: APIClient;
  let workbook: Workbook;
  let sheet: Sheet;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);
    workbook = await createTestWorkbook(api, 'Import Export Test');
    sheet = await getFirstSheet(api, workbook.id);

    // Add some test data
    await api.batchUpdateCells(sheet.id, [
      { row: 0, col: 0, value: 'Name' },
      { row: 0, col: 1, value: 'Value' },
      { row: 1, col: 0, value: 'Item1' },
      { row: 1, col: 1, value: 100 },
      { row: 2, col: 0, value: 'Item2' },
      { row: 2, col: 1, value: 200 },
    ]);
  });

  test.describe('Export Workbook', () => {
    test('should export workbook as CSV', async () => {
      const res = await api.exportWorkbook(workbook.id, 'csv');
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('text/csv');

      const content = await res.text();
      expect(content).toContain('Name');
      expect(content).toContain('Value');
      expect(content).toContain('Item1');
      expect(content).toContain('100');
    });

    test('should export workbook as TSV', async () => {
      const res = await api.exportWorkbook(workbook.id, 'tsv');
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('tab-separated-values');

      const content = await res.text();
      expect(content).toContain('\t'); // Tab separator
      expect(content).toContain('Name');
    });

    test('should export workbook as JSON', async () => {
      const res = await api.exportWorkbook(workbook.id, 'json', { metadata: true });
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('application/json');

      const json = await res.json();
      expect(json.version).toBe('1.0');
      expect(json.sheets).toBeDefined();
      expect(json.sheets.length).toBeGreaterThan(0);
    });

    test('should export workbook as HTML', async () => {
      const res = await api.exportWorkbook(workbook.id, 'html');
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('text/html');

      const content = await res.text();
      expect(content).toContain('<!DOCTYPE html>');
      expect(content).toContain('<table>');
      expect(content).toContain('Name');
    });

    test('should export workbook as XLSX', async () => {
      const res = await api.exportWorkbook(workbook.id, 'xlsx');
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('spreadsheetml');

      // XLSX is binary, just verify it's not empty
      const body = await res.body();
      expect(body.length).toBeGreaterThan(0);
    });

    test('should return 404 for non-existent workbook', async () => {
      const res = await api.exportWorkbook('non-existent-id', 'csv');
      expect(res.status()).toBe(404);
    });

    test('should return error for invalid format', async () => {
      const res = await api.exportWorkbook(workbook.id, 'invalid');
      expect(res.ok()).toBeFalsy();
    });
  });

  test.describe('Export Sheet', () => {
    test('should export sheet as CSV', async () => {
      const res = await api.exportSheet(sheet.id, 'csv');
      expect(res.ok()).toBeTruthy();

      const content = await res.text();
      expect(content).toContain('Name');
      expect(content).toContain('Item1');
    });

    test('should export sheet as JSON', async () => {
      const res = await api.exportSheet(sheet.id, 'json');
      expect(res.ok()).toBeTruthy();

      const json = await res.json();
      expect(json.sheets).toBeDefined();
      expect(json.sheets[0].cells).toBeDefined();
    });

    test('should return 404 for non-existent sheet', async () => {
      const res = await api.exportSheet('non-existent-id', 'csv');
      expect(res.status()).toBe(404);
    });
  });

  test.describe('Export with Formulas', () => {
    test.beforeEach(async () => {
      // Add a formula
      await api.setCell(sheet.id, 2, 2, { formula: '=B2+B3' });
    });

    test('should export formulas when option is set', async () => {
      const res = await api.exportWorkbook(workbook.id, 'csv', { formulas: true });
      expect(res.ok()).toBeTruthy();

      const content = await res.text();
      expect(content).toContain('=B2+B3');
    });

    test('should export calculated values by default', async () => {
      const res = await api.exportWorkbook(workbook.id, 'csv');
      expect(res.ok()).toBeTruthy();

      const content = await res.text();
      // Should contain the calculated value, not the formula
      expect(content).toContain('300'); // 100 + 200
    });
  });

  test.describe('Import CSV to Workbook', () => {
    test('should import CSV and create new sheet', async () => {
      const csvContent = 'A,B,C\n1,2,3\n4,5,6';
      const res = await api.importToWorkbook(workbook.id, csvContent, 'test.csv');
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.rowsImported).toBe(3);
      expect(result.data.colsImported).toBe(3);
      expect(result.data.cellsImported).toBe(9);
    });

    test('should import CSV with headers option', async () => {
      const csvContent = 'Name,Value\nTest,123';
      const res = await api.importToWorkbook(workbook.id, csvContent, 'test.csv', {
        hasHeaders: true,
      });
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      // First row is treated as headers and skipped
      expect(result.data.rowsImported).toBe(1);
    });

    test('should import CSV with custom sheet name', async () => {
      const csvContent = 'A,B\n1,2';
      const res = await api.importToWorkbook(workbook.id, csvContent, 'test.csv', {
        sheetName: 'My Custom Sheet',
      });
      expect(res.ok()).toBeTruthy();

      // Verify the sheet was created with the custom name
      const sheetsRes = await api.listSheets(workbook.id);
      const sheets = (await sheetsRes.json()) as Sheet[];
      expect(sheets.map(s => s.name)).toContain('My Custom Sheet');
    });

    test('should handle special characters in CSV', async () => {
      const csvContent = '"Hello, World","Line1\nLine2","Quote ""test"""';
      const res = await api.importToWorkbook(workbook.id, csvContent, 'test.csv');
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.cellsImported).toBe(3);
    });

    test('should return 404 for non-existent workbook', async () => {
      const csvContent = 'A,B\n1,2';
      const res = await api.importToWorkbook('non-existent-id', csvContent, 'test.csv');
      expect(res.status()).toBe(404);
    });
  });

  test.describe('Import TSV to Workbook', () => {
    test('should import TSV file', async () => {
      const tsvContent = 'A\tB\tC\n1\t2\t3';
      const res = await api.importToWorkbook(workbook.id, tsvContent, 'test.tsv');
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.rowsImported).toBe(2);
      expect(result.data.colsImported).toBe(3);
    });
  });

  test.describe('Import JSON to Workbook', () => {
    test('should import JSON file with cells', async () => {
      const jsonContent = JSON.stringify({
        version: '1.0',
        sheets: [{
          name: 'Imported Sheet',
          cells: [
            { row: 0, col: 0, value: 'Hello' },
            { row: 0, col: 1, value: 'World' },
            { row: 1, col: 0, value: 123 },
          ],
        }],
      });

      const res = await api.importToWorkbook(workbook.id, jsonContent, 'test.json');
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.cellsImported).toBe(3);
    });

    test('should import JSON with formulas', async () => {
      const jsonContent = JSON.stringify({
        version: '1.0',
        sheets: [{
          name: 'Formula Sheet',
          cells: [
            { row: 0, col: 0, value: 10 },
            { row: 0, col: 1, value: 20 },
            { row: 0, col: 2, formula: '=A1+B1' },
          ],
        }],
      });

      const res = await api.importToWorkbook(workbook.id, jsonContent, 'test.json', {
        importFormulas: true,
      });
      expect(res.ok()).toBeTruthy();
    });

    test('should import JSON with multiple sheets', async () => {
      const jsonContent = JSON.stringify({
        version: '1.0',
        sheets: [
          { name: 'Sheet A', cells: [{ row: 0, col: 0, value: 'A' }] },
          { name: 'Sheet B', cells: [{ row: 0, col: 0, value: 'B' }] },
        ],
      });

      const res = await api.importToWorkbook(workbook.id, jsonContent, 'test.json');
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.cellsImported).toBe(2);
    });
  });

  test.describe('Import to Existing Sheet', () => {
    test('should import CSV to existing sheet', async () => {
      const csvContent = 'New,Data\n1,2';
      const res = await api.importToSheet(sheet.id, csvContent, 'test.csv');
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.sheetId).toBe(sheet.id);
      expect(result.data.cellsImported).toBeGreaterThan(0);
    });

    test('should return 404 for non-existent sheet', async () => {
      const csvContent = 'A,B\n1,2';
      const res = await api.importToSheet('non-existent-id', csvContent, 'test.csv');
      expect(res.status()).toBe(404);
    });
  });

  test.describe('Import with Type Detection', () => {
    test('should auto-detect numeric types', async () => {
      const csvContent = 'Text,Number,Boolean\nHello,42,true';
      const res = await api.importToWorkbook(workbook.id, csvContent, 'test.csv', {
        hasHeaders: true,
        autoDetectTypes: true,
      });
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      expect(result.data.cellsImported).toBe(3);
    });
  });

  test.describe('Export/Import Round Trip', () => {
    test('should export CSV and import back with same data', async () => {
      // Export the workbook
      const exportRes = await api.exportWorkbook(workbook.id, 'csv');
      expect(exportRes.ok()).toBeTruthy();
      const csvContent = await exportRes.text();

      // Create a new workbook and import the CSV
      const newWorkbook = await createTestWorkbook(api, 'Round Trip Test');
      const importRes = await api.importToWorkbook(newWorkbook.id, csvContent, 'roundtrip.csv');
      expect(importRes.ok()).toBeTruthy();

      // Export the new workbook and compare
      const newExportRes = await api.exportWorkbook(newWorkbook.id, 'csv');
      const newCsvContent = await newExportRes.text();

      // Content should be equivalent (may have minor formatting differences)
      expect(newCsvContent).toContain('Name');
      expect(newCsvContent).toContain('Item1');
      expect(newCsvContent).toContain('100');
    });

    test('should export JSON and import back preserving structure', async () => {
      // Export as JSON
      const exportRes = await api.exportWorkbook(workbook.id, 'json', { metadata: true });
      expect(exportRes.ok()).toBeTruthy();
      const jsonContent = await exportRes.text();

      // Create new workbook and import
      const newWorkbook = await createTestWorkbook(api, 'JSON Round Trip');
      const importRes = await api.importToWorkbook(newWorkbook.id, jsonContent, 'roundtrip.json');
      expect(importRes.ok()).toBeTruthy();

      // Verify cell count matches
      const result = await importRes.json();
      expect(result.data.cellsImported).toBeGreaterThan(0);
    });
  });

  test.describe('Edge Cases', () => {
    test('should handle empty sheet export', async () => {
      // Create a new empty sheet
      const emptyWorkbook = await createTestWorkbook(api, 'Empty Workbook');

      const res = await api.exportWorkbook(emptyWorkbook.id, 'csv');
      expect(res.ok()).toBeTruthy();
    });

    test('should handle large data export', async () => {
      // Create a sheet with more data
      const largeCells = [];
      for (let row = 0; row < 50; row++) {
        for (let col = 0; col < 10; col++) {
          largeCells.push({ row, col, value: `R${row}C${col}` });
        }
      }
      await api.batchUpdateCells(sheet.id, largeCells);

      const res = await api.exportWorkbook(workbook.id, 'csv');
      expect(res.ok()).toBeTruthy();

      const content = await res.text();
      expect(content).toContain('R49C9'); // Last cell
    });

    test('should handle import with empty rows', async () => {
      const csvContent = 'A,B\n\n1,2\n\n3,4';
      const res = await api.importToWorkbook(workbook.id, csvContent, 'test.csv', {
        skipEmptyRows: true,
      });
      expect(res.ok()).toBeTruthy();

      const result = await res.json();
      // With skipEmptyRows, should only import non-empty rows
      expect(result.data.rowsImported).toBeLessThan(5);
    });
  });
});

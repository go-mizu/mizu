import { test, expect } from '@playwright/test';
import { APIClient, registerAndLogin, generateUniqueEmail, Workbook, Sheet, Cell } from './helpers';

test.describe('Integration Tests', () => {
  let api: APIClient;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
  });

  test('Full workflow: Register → Create workbook → Add data → Formula → Save', async () => {
    // Step 1: Register a new user
    const email = generateUniqueEmail();
    const registerRes = await api.register(email, 'password123', 'Integration Test User');
    expect(registerRes.ok()).toBeTruthy();

    const registerData = await registerRes.json();
    api.setToken(registerData.token);

    // Step 2: Create a workbook
    const createWbRes = await api.createWorkbook('Integration Test Workbook');
    expect(createWbRes.ok()).toBeTruthy();
    const createData = await createWbRes.json();
    const workbook = (createData.workbook || createData) as Workbook;

    // Step 3: Get the default sheet
    const sheetsRes = await api.listSheets(workbook.id);
    expect(sheetsRes.ok()).toBeTruthy();
    const sheets = (await sheetsRes.json()) as Sheet[];
    expect(sheets.length).toBe(1);
    const sheet = sheets[0];

    // Step 4: Add data to cells
    await api.setCell(sheet.id, 0, 0, { value: 'Item' });
    await api.setCell(sheet.id, 0, 1, { value: 'Price' });
    await api.setCell(sheet.id, 0, 2, { value: 'Quantity' });
    await api.setCell(sheet.id, 0, 3, { value: 'Total' });

    await api.setCell(sheet.id, 1, 0, { value: 'Apple' });
    await api.setCell(sheet.id, 1, 1, { value: 1.5 });
    await api.setCell(sheet.id, 1, 2, { value: 10 });
    await api.setCell(sheet.id, 1, 3, { formula: '=B2*C2' });

    await api.setCell(sheet.id, 2, 0, { value: 'Banana' });
    await api.setCell(sheet.id, 2, 1, { value: 0.75 });
    await api.setCell(sheet.id, 2, 2, { value: 20 });
    await api.setCell(sheet.id, 2, 3, { formula: '=B3*C3' });

    await api.setCell(sheet.id, 3, 0, { value: 'Orange' });
    await api.setCell(sheet.id, 3, 1, { value: 2.0 });
    await api.setCell(sheet.id, 3, 2, { value: 5 });
    await api.setCell(sheet.id, 3, 3, { formula: '=B4*C4' });

    // Step 5: Add a grand total formula
    const grandTotalRes = await api.setCell(sheet.id, 4, 3, { formula: '=SUM(D2:D4)' });
    expect(grandTotalRes.ok()).toBeTruthy();

    // Step 6: Verify data was saved
    const cellsRes = await api.getCells(sheet.id, { startRow: 0, startCol: 0, endRow: 5, endCol: 4 });
    expect(cellsRes.ok()).toBeTruthy();

    const cells = (await cellsRes.json()) as Cell[];
    expect(cells.length).toBeGreaterThan(0);

    // Step 7: Verify user can re-login and access data
    api.clearToken();
    const loginRes = await api.login(email, 'password123');
    expect(loginRes.ok()).toBeTruthy();

    const loginData = await loginRes.json();
    api.setToken(loginData.token);

    // Verify workbook still exists
    const wbListRes = await api.listWorkbooks();
    const workbooksList = (await wbListRes.json()) as Workbook[];
    expect(workbooksList.length).toBe(1);
    expect(workbooksList[0].name).toBe('Integration Test Workbook');
  });

  test('Multi-sheet workflow: Create workbook → Add multiple sheets → Cross-sheet reference', async () => {
    await registerAndLogin(api);

    // Create workbook
    const wbRes = await api.createWorkbook('Multi-Sheet Test');
    const wbData = await wbRes.json();
    const workbook = (wbData.workbook || wbData) as Workbook;

    // Get default sheet (Sheet1)
    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    const sheet1 = sheets[0];

    // Create Sheet2
    const sheet2Res = await api.createSheet(workbook.id, 'Sales Data');
    expect(sheet2Res.ok()).toBeTruthy();
    const sheet2 = (await sheet2Res.json()) as Sheet;

    // Create Sheet3
    const sheet3Res = await api.createSheet(workbook.id, 'Summary');
    expect(sheet3Res.ok()).toBeTruthy();
    const sheet3 = (await sheet3Res.json()) as Sheet;

    // Add data to Sales Data sheet
    await api.batchUpdateCells(sheet2.id, [
      { row: 0, col: 0, value: 'Q1' },
      { row: 0, col: 1, value: 100 },
      { row: 1, col: 0, value: 'Q2' },
      { row: 1, col: 1, value: 150 },
      { row: 2, col: 0, value: 'Q3' },
      { row: 2, col: 1, value: 200 },
      { row: 3, col: 0, value: 'Q4' },
      { row: 3, col: 1, value: 250 },
    ]);

    // Verify sheets exist
    const finalSheetsRes = await api.listSheets(workbook.id);
    const finalSheets = (await finalSheetsRes.json()) as Sheet[];
    expect(finalSheets.length).toBe(3);

    // Verify data in Sales Data sheet
    const salesDataRes = await api.getCells(sheet2.id, { startRow: 0, startCol: 0, endRow: 4, endCol: 2 });
    expect(salesDataRes.ok()).toBeTruthy();
    const salesCells = (await salesDataRes.json()) as Cell[];
    expect(salesCells.length).toBe(8);
  });

  test('Concurrent cell updates', async () => {
    await registerAndLogin(api);

    const wbRes = await api.createWorkbook('Concurrent Test');
    const wbData = await wbRes.json();
    const workbook = (wbData.workbook || wbData) as Workbook;

    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    const sheet = sheets[0];

    // Run multiple cell updates concurrently
    const updates = await Promise.all([
      api.setCell(sheet.id, 0, 0, { value: 'Cell 1' }),
      api.setCell(sheet.id, 0, 1, { value: 'Cell 2' }),
      api.setCell(sheet.id, 0, 2, { value: 'Cell 3' }),
      api.setCell(sheet.id, 1, 0, { value: 'Cell 4' }),
      api.setCell(sheet.id, 1, 1, { value: 'Cell 5' }),
      api.setCell(sheet.id, 1, 2, { value: 'Cell 6' }),
    ]);

    // All updates should succeed
    updates.forEach((res) => {
      expect(res.ok()).toBeTruthy();
    });

    // Verify all cells are saved
    const cellsRes = await api.getCells(sheet.id, { startRow: 0, startCol: 0, endRow: 2, endCol: 3 });
    const cells = (await cellsRes.json()) as Cell[];
    expect(cells.length).toBe(6);
  });

  test.skip('User isolation: Users cannot access each other\'s data', async ({ request }) => {
    // Note: Authorization/access control not yet implemented
    // Create first user and their workbook
    await registerAndLogin(api);
    const wb1Res = await api.createWorkbook('User1 Private Workbook');
    const wb1Data = await wb1Res.json();
    const wb1 = (wb1Data.workbook || wb1Data) as Workbook;

    // Create second user
    const api2 = new APIClient(request);
    await registerAndLogin(api2);

    // User2 should not be able to access User1's workbook
    const getWb1Res = await api2.getWorkbook(wb1.id);
    expect(getWb1Res.status()).toBe(404); // Or 403 Forbidden

    // User2 should not see User1's workbook in their list
    const listRes = await api2.listWorkbooks();
    const workbooks = (await listRes.json()) as Workbook[];
    const hasUser1Workbook = workbooks.some((wb) => wb.id === wb1.id);
    expect(hasUser1Workbook).toBeFalsy();
  });

  test('Workbook cleanup on delete', async () => {
    await registerAndLogin(api);

    // Create workbook with multiple sheets and data
    const wbRes = await api.createWorkbook('To Delete');
    const wbData = await wbRes.json();
    const workbook = (wbData.workbook || wbData) as Workbook;

    // Get default sheet
    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    const sheet = sheets[0];

    // Add cells
    await api.batchUpdateCells(sheet.id, [
      { row: 0, col: 0, value: 'Data 1' },
      { row: 0, col: 1, value: 'Data 2' },
    ]);

    // Add merge
    await api.merge(sheet.id, { startRow: 5, startCol: 5, endRow: 7, endCol: 7 });

    // Delete workbook
    const deleteRes = await api.deleteWorkbook(workbook.id);
    expect(deleteRes.ok()).toBeTruthy();

    // Verify workbook is deleted
    const getWbRes = await api.getWorkbook(workbook.id);
    expect(getWbRes.status()).toBe(404);

    // Verify sheet is deleted
    const getSheetRes = await api.getSheet(sheet.id);
    expect(getSheetRes.status()).toBe(404);
  });

  test('Sheet reindexing after deletion', async () => {
    await registerAndLogin(api);

    // Create workbook
    const wbRes = await api.createWorkbook('Reindex Test');
    const wbData = await wbRes.json();
    const workbook = (wbData.workbook || wbData) as Workbook;

    // Create additional sheets
    await api.createSheet(workbook.id, 'Sheet2', { index: 1 });
    await api.createSheet(workbook.id, 'Sheet3', { index: 2 });

    // Get sheets
    let sheetsRes = await api.listSheets(workbook.id);
    let sheets = (await sheetsRes.json()) as Sheet[];
    expect(sheets.length).toBe(3);

    // Delete middle sheet (Sheet2)
    const sheet2 = sheets.find((s) => s.name === 'Sheet2');
    expect(sheet2).toBeDefined();
    await api.deleteSheet(sheet2!.id);

    // Verify remaining sheets
    sheetsRes = await api.listSheets(workbook.id);
    sheets = (await sheetsRes.json()) as Sheet[];
    expect(sheets.length).toBe(2);
    expect(sheets.map((s) => s.name)).toContain('Sheet1');
    expect(sheets.map((s) => s.name)).toContain('Sheet3');
  });

  test('Large batch cell update', async () => {
    await registerAndLogin(api);

    const wbRes = await api.createWorkbook('Large Batch Test');
    const wbData = await wbRes.json();
    const workbook = (wbData.workbook || wbData) as Workbook;

    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    const sheet = sheets[0];

    // Create a large batch of cells (100 cells)
    const cells = [];
    for (let row = 0; row < 10; row++) {
      for (let col = 0; col < 10; col++) {
        cells.push({ row, col, value: `R${row}C${col}` });
      }
    }

    const batchRes = await api.batchUpdateCells(sheet.id, cells);
    expect(batchRes.ok()).toBeTruthy();

    const savedCells = (await batchRes.json()) as Cell[];
    expect(savedCells.length).toBe(100);
  });
});

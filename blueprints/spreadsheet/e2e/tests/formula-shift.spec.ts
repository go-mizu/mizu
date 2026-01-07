import { test, expect } from '@playwright/test';
import { APIClient, registerAndLogin, Sheet, Cell } from './helpers';

test.describe('Formula Shifting Tests', () => {
  let api: APIClient;
  let sheetId: string;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);

    // Create workbook and get sheet
    const wbRes = await api.createWorkbook('Formula Shift Test');
    const wbData = await wbRes.json();
    const workbook = wbData.workbook || wbData;

    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    sheetId = sheets[0].id;
  });

  test('Insert row shifts formula references down', async () => {
    // Setup: Create cells with formulas
    await api.setCell(sheetId, 0, 0, { value: 10 }); // A1 = 10
    await api.setCell(sheetId, 1, 0, { value: 20 }); // A2 = 20
    await api.setCell(sheetId, 2, 0, { value: 30 }); // A3 = 30
    await api.setCell(sheetId, 3, 0, { formula: '=SUM(A1:A3)' }); // A4 = SUM(A1:A3)
    await api.setCell(sheetId, 4, 0, { formula: '=A3*2' }); // A5 = A3*2

    // Insert a row at index 1 (between A1 and A2)
    const insertRes = await api.insertRows(sheetId, 1, 1);
    expect(insertRes.ok()).toBeTruthy();

    // Verify formulas were shifted
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 6, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    // Find the SUM formula cell (now at row 4)
    const sumCell = cells.find(c => c.row === 4 && c.col === 0);
    expect(sumCell?.formula).toBe('=SUM(A1:A4)'); // Range expanded

    // Find the multiplication formula (now at row 5)
    const multCell = cells.find(c => c.row === 5 && c.col === 0);
    expect(multCell?.formula).toBe('=A4*2'); // Reference shifted
  });

  test('Delete row shifts formula references up and marks deleted refs as #REF!', async () => {
    // Setup: Create cells with formulas
    await api.setCell(sheetId, 0, 0, { value: 10 }); // A1 = 10
    await api.setCell(sheetId, 1, 0, { value: 20 }); // A2 = 20
    await api.setCell(sheetId, 2, 0, { value: 30 }); // A3 = 30
    await api.setCell(sheetId, 3, 0, { formula: '=A2' }); // A4 = A2 (references the row to be deleted)
    await api.setCell(sheetId, 4, 0, { formula: '=A3*2' }); // A5 = A3*2

    // Delete row at index 1 (A2)
    const deleteRes = await api.deleteRows(sheetId, 1, 1);
    expect(deleteRes.ok()).toBeTruthy();

    // Verify formulas were updated
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 4, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    // The formula that referenced deleted row should have #REF!
    const refCell = cells.find(c => c.row === 2 && c.col === 0);
    expect(refCell?.formula).toContain('#REF!');

    // The formula that referenced A3 should now reference A2
    const shiftedCell = cells.find(c => c.row === 3 && c.col === 0);
    expect(shiftedCell?.formula).toBe('=A2*2');
  });

  test('Insert column shifts formula column references', async () => {
    // Setup: Create cells with formulas
    await api.setCell(sheetId, 0, 0, { value: 10 }); // A1
    await api.setCell(sheetId, 0, 1, { value: 20 }); // B1
    await api.setCell(sheetId, 0, 2, { value: 30 }); // C1
    await api.setCell(sheetId, 0, 3, { formula: '=SUM(A1:C1)' }); // D1 = SUM(A1:C1)
    await api.setCell(sheetId, 0, 4, { formula: '=C1*2' }); // E1 = C1*2

    // Insert a column at index 1 (between A and B)
    const insertRes = await api.insertCols(sheetId, 1, 1);
    expect(insertRes.ok()).toBeTruthy();

    // Verify formulas were shifted
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 1, endCol: 6 });
    const cells = (await cellsRes.json()) as Cell[];

    // Find the SUM formula cell (now at col 4)
    const sumCell = cells.find(c => c.row === 0 && c.col === 4);
    expect(sumCell?.formula).toBe('=SUM(A1:D1)'); // Range expanded

    // Find the multiplication formula (now at col 5)
    const multCell = cells.find(c => c.row === 0 && c.col === 5);
    expect(multCell?.formula).toBe('=D1*2'); // Reference shifted
  });

  test('Absolute references are not shifted', async () => {
    // Setup: Create cells with absolute reference formulas
    await api.setCell(sheetId, 0, 0, { value: 100 }); // A1 = 100
    await api.setCell(sheetId, 1, 0, { value: 10 }); // A2 = 10
    await api.setCell(sheetId, 2, 0, { formula: '=$A$1*A2' }); // A3 = $A$1*A2

    // Insert a row at index 1
    const insertRes = await api.insertRows(sheetId, 1, 1);
    expect(insertRes.ok()).toBeTruthy();

    // Verify the absolute reference was preserved
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 4, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    // Find the formula cell (now at row 3)
    const formulaCell = cells.find(c => c.row === 3 && c.col === 0);
    expect(formulaCell?.formula).toBe('=$A$1*A3'); // $A$1 stays, A2 becomes A3
  });

  test('Range formulas expand correctly on row insert', async () => {
    // Setup: Create a SUM formula over a range
    await api.setCell(sheetId, 0, 0, { value: 10 });
    await api.setCell(sheetId, 1, 0, { value: 20 });
    await api.setCell(sheetId, 2, 0, { value: 30 });
    await api.setCell(sheetId, 3, 0, { value: 40 });
    await api.setCell(sheetId, 4, 0, { formula: '=SUM(A1:A4)' });

    // Insert a row in the middle of the range (at index 2)
    const insertRes = await api.insertRows(sheetId, 2, 1);
    expect(insertRes.ok()).toBeTruthy();

    // Verify the range expanded
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 6, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    // Find the SUM formula (now at row 5)
    const sumCell = cells.find(c => c.row === 5 && c.col === 0);
    expect(sumCell?.formula).toBe('=SUM(A1:A5)'); // Range expanded to include new row
  });

  test('Multiple row delete shifts formulas correctly', async () => {
    // Setup: Create cells
    for (let i = 0; i < 10; i++) {
      await api.setCell(sheetId, i, 0, { value: (i + 1) * 10 });
    }
    await api.setCell(sheetId, 10, 0, { formula: '=SUM(A1:A10)' });
    await api.setCell(sheetId, 11, 0, { formula: '=A7+A8' }); // References rows to be deleted

    // Delete rows 4-6 (3 rows)
    const deleteRes = await api.deleteRows(sheetId, 4, 3);
    expect(deleteRes.ok()).toBeTruthy();

    // Verify
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 10, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    // SUM formula should have contracted range
    const sumCell = cells.find(c => c.formula?.includes('SUM'));
    expect(sumCell?.formula).toBe('=SUM(A1:A7)');

    // Formula referencing deleted rows should have #REF!
    const refCell = cells.find(c => c.formula?.includes('#REF!'));
    expect(refCell).toBeDefined();
  });
});

test.describe('Copy Range with Formula Adjustment', () => {
  let api: APIClient;
  let sheetId: string;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);

    const wbRes = await api.createWorkbook('Copy Test');
    const wbData = await wbRes.json();
    const workbook = wbData.workbook || wbData;

    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    sheetId = sheets[0].id;
  });

  test('Copy formula adjusts relative references', async () => {
    // Setup: Create a formula in A1
    await api.setCell(sheetId, 0, 0, { value: 10 });
    await api.setCell(sheetId, 0, 1, { value: 20 });
    await api.setCell(sheetId, 0, 2, { formula: '=A1+B1' });

    // Copy the formula from C1 to C2
    const copyRes = await api.copyRange(sheetId, {
      sourceRange: { startRow: 0, startCol: 2, endRow: 0, endCol: 2 },
      destRow: 1,
      destCol: 2,
    });
    expect(copyRes.ok()).toBeTruthy();

    // Verify the formula was adjusted
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 2, endCol: 3 });
    const cells = (await cellsRes.json()) as Cell[];

    const copiedCell = cells.find(c => c.row === 1 && c.col === 2);
    expect(copiedCell?.formula).toBe('=A2+B2'); // References shifted down
  });

  test('Copy formula preserves absolute references', async () => {
    // Setup: Create a formula with mixed references
    await api.setCell(sheetId, 0, 0, { value: 100 }); // Tax rate
    await api.setCell(sheetId, 1, 0, { value: 50 });
    await api.setCell(sheetId, 1, 1, { formula: '=A2*$A$1' }); // Price * tax rate

    // Copy to row 2
    const copyRes = await api.copyRange(sheetId, {
      sourceRange: { startRow: 1, startCol: 1, endRow: 1, endCol: 1 },
      destRow: 2,
      destCol: 1,
    });
    expect(copyRes.ok()).toBeTruthy();

    // Verify
    const cellsRes = await api.getCells(sheetId, { startRow: 0, startCol: 0, endRow: 3, endCol: 2 });
    const cells = (await cellsRes.json()) as Cell[];

    const copiedCell = cells.find(c => c.row === 2 && c.col === 1);
    expect(copiedCell?.formula).toBe('=A3*$A$1'); // A2 becomes A3, $A$1 stays
  });
});

test.describe('Cross-Sheet References', () => {
  let api: APIClient;
  let workbookId: string;
  let sheet1Id: string;
  let sheet2Id: string;

  test.beforeEach(async ({ request }) => {
    api = new APIClient(request);
    await registerAndLogin(api);

    const wbRes = await api.createWorkbook('Cross-Sheet Test');
    const wbData = await wbRes.json();
    const workbook = wbData.workbook || wbData;
    workbookId = workbook.id;

    // Get first sheet
    const sheetsRes = await api.listSheets(workbook.id);
    const sheets = (await sheetsRes.json()) as Sheet[];
    sheet1Id = sheets[0].id;

    // Create second sheet
    const sheet2Res = await api.createSheet(workbook.id, 'Data');
    const sheet2 = (await sheet2Res.json()) as Sheet;
    sheet2Id = sheet2.id;
  });

  test('Cross-sheet formula evaluates correctly', async () => {
    // Set value in Sheet1
    await api.setCell(sheet1Id, 0, 0, { value: 100 });

    // Reference it from Data sheet
    const setRes = await api.setCell(sheet2Id, 0, 0, { formula: "='Sheet1'!A1" });
    expect(setRes.ok()).toBeTruthy();

    // Verify the formula evaluated
    const cellsRes = await api.getCells(sheet2Id, { startRow: 0, startCol: 0, endRow: 1, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    const cell = cells.find(c => c.row === 0 && c.col === 0);
    expect(cell?.display).toBe('100');
  });

  test('Cross-sheet range formula evaluates correctly', async () => {
    // Set values in Sheet1
    await api.setCell(sheet1Id, 0, 0, { value: 10 });
    await api.setCell(sheet1Id, 1, 0, { value: 20 });
    await api.setCell(sheet1Id, 2, 0, { value: 30 });

    // Sum from Data sheet
    const setRes = await api.setCell(sheet2Id, 0, 0, { formula: "=SUM('Sheet1'!A1:A3)" });
    expect(setRes.ok()).toBeTruthy();

    // Verify
    const cellsRes = await api.getCells(sheet2Id, { startRow: 0, startCol: 0, endRow: 1, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    const cell = cells.find(c => c.row === 0 && c.col === 0);
    expect(cell?.display).toBe('60');
  });

  test('Cross-sheet references not affected by local row insert', async () => {
    // Set value in Sheet1
    await api.setCell(sheet1Id, 0, 0, { value: 100 });

    // Reference from Data sheet
    await api.setCell(sheet2Id, 0, 0, { formula: "='Sheet1'!A1" });

    // Insert row in Data sheet (should not affect cross-sheet reference)
    const insertRes = await api.insertRows(sheet2Id, 0, 1);
    expect(insertRes.ok()).toBeTruthy();

    // Verify formula still references Sheet1!A1
    const cellsRes = await api.getCells(sheet2Id, { startRow: 0, startCol: 0, endRow: 2, endCol: 1 });
    const cells = (await cellsRes.json()) as Cell[];

    const cell = cells.find(c => c.formula?.includes('Sheet1'));
    expect(cell?.formula).toBe("='Sheet1'!A1"); // Should not be shifted
  });
});

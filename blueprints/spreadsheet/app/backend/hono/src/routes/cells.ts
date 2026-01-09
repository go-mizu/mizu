import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import {
  BatchCellUpdateSchema,
  CreateMergeSchema,
  InsertRowsSchema,
  DeleteRowsSchema,
  InsertColsSchema,
  DeleteColsSchema,
} from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';
import { verifySheetAccess } from './sheets.js';

const cells = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All cell routes require authentication
cells.use('*', authRequired);

/**
 * GET /sheets/:sheetId/cells - Get all cells for a sheet
 */
cells.get('/:sheetId/cells', async (c) => {
  const { sheetId } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const cellList = await db.getCellsBySheet(sheetId);

  // Transform to API format
  const cellsMap: Record<string, {
    value: string | null;
    formula: string | null;
    display: string | null;
    format: string | null;
  }> = {};

  for (const cell of cellList) {
    const key = `${cell.row_num},${cell.col_num}`;
    cellsMap[key] = {
      value: cell.value,
      formula: cell.formula,
      display: cell.display,
      format: cell.format,
    };
  }

  return c.json({ cells: cellsMap });
});

/**
 * PUT /sheets/:sheetId/cells - Batch update cells
 */
cells.put('/:sheetId/cells', zValidator('json', BatchCellUpdateSchema), async (c) => {
  const { sheetId } = c.req.param();
  const { cells: cellUpdates } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const inputs = cellUpdates.map(cell => ({
    id: ulid(),
    sheet_id: sheetId,
    row_num: cell.row,
    col_num: cell.col,
    value: cell.value ?? null,
    formula: cell.formula ?? null,
    display: cell.display ?? null,
    format: cell.format ?? null,
  }));

  const updated = await db.upsertCells(inputs);

  return c.json({
    cells: updated.map(cell => ({
      row: cell.row_num,
      col: cell.col_num,
      value: cell.value,
      formula: cell.formula,
      display: cell.display,
      format: cell.format,
    })),
  });
});

/**
 * GET /sheets/:sheetId/cells/:row/:col - Get single cell
 */
cells.get('/:sheetId/cells/:row/:col', async (c) => {
  const { sheetId, row, col } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const rowNum = parseInt(row, 10);
  const colNum = parseInt(col, 10);

  if (isNaN(rowNum) || isNaN(colNum) || rowNum < 0 || colNum < 0) {
    throw ApiError.badRequest('Invalid row or column number');
  }

  const cell = await db.getCell(sheetId, rowNum, colNum);

  return c.json({
    cell: cell ? {
      row: cell.row_num,
      col: cell.col_num,
      value: cell.value,
      formula: cell.formula,
      display: cell.display,
      format: cell.format,
    } : null,
  });
});

/**
 * PUT /sheets/:sheetId/cells/:row/:col - Update single cell
 */
const SingleCellUpdateSchema = z.object({
  value: z.string().nullable().optional(),
  formula: z.string().nullable().optional(),
  display: z.string().nullable().optional(),
  format: z.string().nullable().optional(),
});

cells.put('/:sheetId/cells/:row/:col', zValidator('json', SingleCellUpdateSchema), async (c) => {
  const { sheetId, row, col } = c.req.param();
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const rowNum = parseInt(row, 10);
  const colNum = parseInt(col, 10);

  if (isNaN(rowNum) || isNaN(colNum) || rowNum < 0 || colNum < 0) {
    throw ApiError.badRequest('Invalid row or column number');
  }

  const cell = await db.upsertCell({
    id: ulid(),
    sheet_id: sheetId,
    row_num: rowNum,
    col_num: colNum,
    ...data,
  });

  return c.json({
    cell: {
      row: cell.row_num,
      col: cell.col_num,
      value: cell.value,
      formula: cell.formula,
      display: cell.display,
      format: cell.format,
    },
  });
});

/**
 * DELETE /sheets/:sheetId/cells/:row/:col - Delete cell
 */
cells.delete('/:sheetId/cells/:row/:col', async (c) => {
  const { sheetId, row, col } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const rowNum = parseInt(row, 10);
  const colNum = parseInt(col, 10);

  if (isNaN(rowNum) || isNaN(colNum) || rowNum < 0 || colNum < 0) {
    throw ApiError.badRequest('Invalid row or column number');
  }

  await db.deleteCell(sheetId, rowNum, colNum);
  return c.json({ message: 'Cell deleted' });
});

// ============================================================================
// Merge Operations
// ============================================================================

/**
 * GET /sheets/:sheetId/merges - Get merged regions
 */
cells.get('/:sheetId/merges', async (c) => {
  const { sheetId } = c.req.param();
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  const regions = await db.getMergedRegions(sheetId);
  return c.json({
    merges: regions.map(r => ({
      id: r.id,
      startRow: r.start_row,
      startCol: r.start_col,
      endRow: r.end_row,
      endCol: r.end_col,
    })),
  });
});

/**
 * POST /sheets/:sheetId/merge - Merge cells
 */
cells.post('/:sheetId/merge', zValidator('json', CreateMergeSchema), async (c) => {
  const { sheetId } = c.req.param();
  const data = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  // Validate merge range
  if (data.end_row < data.start_row || data.end_col < data.start_col) {
    throw ApiError.badRequest('Invalid merge range');
  }

  // Check for overlapping merges
  const existing = await db.getMergedRegions(sheetId);
  for (const region of existing) {
    const overlaps =
      data.start_row <= region.end_row &&
      data.end_row >= region.start_row &&
      data.start_col <= region.end_col &&
      data.end_col >= region.start_col;
    if (overlaps) {
      throw ApiError.conflict('Merge range overlaps with existing merge');
    }
  }

  const merge = await db.createMergedRegion({
    id: ulid(),
    sheet_id: sheetId,
    ...data,
  });

  return c.json({
    merge: {
      id: merge.id,
      startRow: merge.start_row,
      startCol: merge.start_col,
      endRow: merge.end_row,
      endCol: merge.end_col,
    },
  }, 201);
});

/**
 * POST /sheets/:sheetId/unmerge - Unmerge cells
 */
const UnmergeSchema = z.object({
  merge_id: z.string(),
});

cells.post('/:sheetId/unmerge', zValidator('json', UnmergeSchema), async (c) => {
  const { sheetId } = c.req.param();
  const { merge_id } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  await db.deleteMergedRegion(merge_id);
  return c.json({ message: 'Cells unmerged' });
});

// ============================================================================
// Row/Column Operations
// ============================================================================

/**
 * POST /sheets/:sheetId/rows/insert - Insert rows
 */
cells.post('/:sheetId/rows/insert', zValidator('json', InsertRowsSchema), async (c) => {
  const { sheetId } = c.req.param();
  const { start_row, count } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  // Shift existing cells down
  await db.shiftCellsDown(sheetId, start_row, count);

  return c.json({ message: `Inserted ${count} row(s) at position ${start_row}` });
});

/**
 * POST /sheets/:sheetId/rows/delete - Delete rows
 */
cells.post('/:sheetId/rows/delete', zValidator('json', DeleteRowsSchema), async (c) => {
  const { sheetId } = c.req.param();
  const { start_row, count } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  // Delete cells in the range
  await db.deleteCellsInRange(sheetId, start_row, start_row + count - 1, 0, 999);

  // Shift remaining cells up
  await db.shiftCellsUp(sheetId, start_row, count);

  return c.json({ message: `Deleted ${count} row(s) starting at position ${start_row}` });
});

/**
 * POST /sheets/:sheetId/cols/insert - Insert columns
 */
cells.post('/:sheetId/cols/insert', zValidator('json', InsertColsSchema), async (c) => {
  const { sheetId } = c.req.param();
  const { start_col, count } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  // Shift existing cells right
  await db.shiftCellsRight(sheetId, start_col, count);

  return c.json({ message: `Inserted ${count} column(s) at position ${start_col}` });
});

/**
 * POST /sheets/:sheetId/cols/delete - Delete columns
 */
cells.post('/:sheetId/cols/delete', zValidator('json', DeleteColsSchema), async (c) => {
  const { sheetId } = c.req.param();
  const { start_col, count } = c.req.valid('json');
  const user = c.get('user');
  const db = c.get('db');

  await verifySheetAccess(db, sheetId, user.id);

  // Delete cells in the range
  await db.deleteCellsInRange(sheetId, 0, 999, start_col, start_col + count - 1);

  // Shift remaining cells left
  await db.shiftCellsLeft(sheetId, start_col, count);

  return c.json({ message: `Deleted ${count} column(s) starting at position ${start_col}` });
});

export { cells };

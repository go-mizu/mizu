/**
 * PostgreSQL tile-based cell storage
 */

import type postgres from 'postgres';
import type { Cell, UpsertCellInput, MergedRegion, CreateMergeInput } from '../../types.js';
import {
  TILE_HEIGHT,
  TILE_WIDTH,
  cellToTile,
  tileCellKey,
  parseTileCellKey,
  tileToCell,
  getTileRange,
  createEmptyTile,
  serializeTile,
  deserializeTile,
  isTileEmpty,
  type Tile,
  type TileCell,
} from '../../tile.js';

export class PostgresTilesStore {
  constructor(private sql: postgres.Sql) {}

  // ============================================================================
  // Tile Operations
  // ============================================================================

  private async loadTile(sheetId: string, tileRow: number, tileCol: number): Promise<Tile> {
    const [row] = await this.sql<[{ values_blob: string } | undefined]>`
      SELECT values_blob FROM sheet_tiles
      WHERE sheet_id = ${sheetId} AND tile_row = ${tileRow} AND tile_col = ${tileCol}
    `;
    return deserializeTile(row?.values_blob);
  }

  private async saveTile(sheetId: string, tileRow: number, tileCol: number, tile: Tile): Promise<void> {
    if (isTileEmpty(tile)) {
      await this.sql`
        DELETE FROM sheet_tiles
        WHERE sheet_id = ${sheetId} AND tile_row = ${tileRow} AND tile_col = ${tileCol}
      `;
      return;
    }

    const blob = serializeTile(tile);
    await this.sql`
      INSERT INTO sheet_tiles (sheet_id, tile_row, tile_col, tile_h, tile_w, encoding, values_blob, updated_at)
      VALUES (${sheetId}, ${tileRow}, ${tileCol}, ${TILE_HEIGHT}, ${TILE_WIDTH}, 'json_v1', ${blob}, NOW())
      ON CONFLICT (sheet_id, tile_row, tile_col) DO UPDATE SET
        values_blob = EXCLUDED.values_blob,
        updated_at = EXCLUDED.updated_at
    `;
  }

  // ============================================================================
  // Cell Operations (public API)
  // ============================================================================

  async getCell(sheetId: string, row: number, col: number): Promise<Cell | null> {
    const pos = cellToTile(row, col);
    const tile = await this.loadTile(sheetId, pos.tileRow, pos.tileCol);
    const key = tileCellKey(pos.offsetRow, pos.offsetCol);
    const tc = tile.cells[key];

    if (!tc) return null;

    return {
      id: tc.id || '',
      sheet_id: sheetId,
      row_num: row,
      col_num: col,
      value: tc.value !== undefined ? tc.value : null,
      formula: tc.formula !== undefined ? tc.formula : null,
      display: tc.display !== undefined ? tc.display : null,
      format: tc.format !== undefined ? tc.format : null,
      created_at: '',
      updated_at: '',
    };
  }

  async getCellsBySheet(sheetId: string): Promise<Cell[]> {
    const rows = await this.sql<Array<{
      tile_row: number;
      tile_col: number;
      values_blob: string;
    }>>`
      SELECT tile_row, tile_col, values_blob FROM sheet_tiles
      WHERE sheet_id = ${sheetId} ORDER BY tile_row, tile_col
    `;

    const cells: Cell[] = [];

    for (const row of rows) {
      const tile = deserializeTile(row.values_blob);
      for (const [key, tc] of Object.entries(tile.cells)) {
        if (!tc) continue;
        const parsed = parseTileCellKey(key);
        if (!parsed) continue;
        const [offsetRow, offsetCol] = parsed;
        const [cellRow, cellCol] = tileToCell(row.tile_row, row.tile_col, offsetRow, offsetCol);

        cells.push({
          id: tc.id || '',
          sheet_id: sheetId,
          row_num: cellRow,
          col_num: cellCol,
          value: tc.value !== undefined ? tc.value : null,
          formula: tc.formula !== undefined ? tc.formula : null,
          display: tc.display !== undefined ? tc.display : null,
          format: tc.format !== undefined ? tc.format : null,
          created_at: '',
          updated_at: '',
        });
      }
    }

    // Sort by row, then col
    cells.sort((a, b) => {
      if (a.row_num !== b.row_num) return a.row_num - b.row_num;
      return a.col_num - b.col_num;
    });

    return cells;
  }

  async upsertCell(input: UpsertCellInput & { id: string }): Promise<Cell> {
    const pos = cellToTile(input.row_num, input.col_num);
    const tile = await this.loadTile(input.sheet_id, pos.tileRow, pos.tileCol);
    const key = tileCellKey(pos.offsetRow, pos.offsetCol);

    const existing = tile.cells[key];
    const tc: TileCell = {
      id: existing?.id || input.id,
      value: input.value !== undefined ? input.value : (existing?.value !== undefined ? existing.value : null),
      formula: input.formula !== undefined ? input.formula : (existing?.formula !== undefined ? existing.formula : null),
      display: input.display !== undefined ? input.display : (existing?.display !== undefined ? existing.display : null),
      format: input.format !== undefined ? input.format : (existing?.format !== undefined ? existing.format : null),
    };

    tile.cells[key] = tc;
    await this.saveTile(input.sheet_id, pos.tileRow, pos.tileCol, tile);

    return {
      id: tc.id || '',
      sheet_id: input.sheet_id,
      row_num: input.row_num,
      col_num: input.col_num,
      value: tc.value !== undefined ? tc.value : null,
      formula: tc.formula !== undefined ? tc.formula : null,
      display: tc.display !== undefined ? tc.display : null,
      format: tc.format !== undefined ? tc.format : null,
      created_at: '',
      updated_at: '',
    };
  }

  async upsertCells(inputs: Array<UpsertCellInput & { id: string }>): Promise<Cell[]> {
    if (inputs.length === 0) return [];

    // Group by tile
    const tileGroups = new Map<string, Array<UpsertCellInput & { id: string }>>();

    for (const input of inputs) {
      const pos = cellToTile(input.row_num, input.col_num);
      const tileKey = `${input.sheet_id}:${pos.tileRow}:${pos.tileCol}`;
      if (!tileGroups.has(tileKey)) {
        tileGroups.set(tileKey, []);
      }
      tileGroups.get(tileKey)!.push(input);
    }

    const results: Cell[] = [];

    for (const [tileKey, groupInputs] of tileGroups) {
      const [sheetId, tileRowStr, tileColStr] = tileKey.split(':');
      const tileRow = parseInt(tileRowStr, 10);
      const tileCol = parseInt(tileColStr, 10);

      const tile = await this.loadTile(sheetId, tileRow, tileCol);

      for (const input of groupInputs) {
        const pos = cellToTile(input.row_num, input.col_num);
        const key = tileCellKey(pos.offsetRow, pos.offsetCol);

        const existing = tile.cells[key];
        const tc: TileCell = {
          id: existing?.id || input.id,
          value: input.value !== undefined ? input.value : (existing?.value !== undefined ? existing.value : null),
          formula: input.formula !== undefined ? input.formula : (existing?.formula !== undefined ? existing.formula : null),
          display: input.display !== undefined ? input.display : (existing?.display !== undefined ? existing.display : null),
          format: input.format !== undefined ? input.format : (existing?.format !== undefined ? existing.format : null),
        };

        tile.cells[key] = tc;

        results.push({
          id: tc.id || '',
          sheet_id: input.sheet_id,
          row_num: input.row_num,
          col_num: input.col_num,
          value: tc.value !== undefined ? tc.value : null,
          formula: tc.formula !== undefined ? tc.formula : null,
          display: tc.display !== undefined ? tc.display : null,
          format: tc.format !== undefined ? tc.format : null,
          created_at: '',
          updated_at: '',
        });
      }

      await this.saveTile(sheetId, tileRow, tileCol, tile);
    }

    return results;
  }

  async deleteCell(sheetId: string, row: number, col: number): Promise<void> {
    const pos = cellToTile(row, col);
    const tile = await this.loadTile(sheetId, pos.tileRow, pos.tileCol);
    const key = tileCellKey(pos.offsetRow, pos.offsetCol);

    delete tile.cells[key];
    await this.saveTile(sheetId, pos.tileRow, pos.tileCol, tile);
  }

  async deleteCellsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    const range = getTileRange(startRow, startCol, endRow, endCol);

    const rows = await this.sql<Array<{
      tile_row: number;
      tile_col: number;
      values_blob: string;
    }>>`
      SELECT tile_row, tile_col, values_blob FROM sheet_tiles
      WHERE sheet_id = ${sheetId}
        AND tile_row >= ${range.startTileRow} AND tile_row <= ${range.endTileRow}
        AND tile_col >= ${range.startTileCol} AND tile_col <= ${range.endTileCol}
    `;

    for (const row of rows) {
      const tile = deserializeTile(row.values_blob);
      let modified = false;

      for (const key of Object.keys(tile.cells)) {
        const parsed = parseTileCellKey(key);
        if (!parsed) continue;
        const [offsetRow, offsetCol] = parsed;
        const [cellRow, cellCol] = tileToCell(row.tile_row, row.tile_col, offsetRow, offsetCol);

        if (cellRow >= startRow && cellRow <= endRow && cellCol >= startCol && cellCol <= endCol) {
          delete tile.cells[key];
          modified = true;
        }
      }

      if (modified) {
        await this.saveTile(sheetId, row.tile_row, row.tile_col, tile);
      }
    }
  }

  // ============================================================================
  // Shift Operations
  // ============================================================================

  async shiftCellsDown(sheetId: string, startRow: number, count: number): Promise<void> {
    await this.shiftCells(sheetId, 'row', startRow, count);
  }

  async shiftCellsUp(sheetId: string, startRow: number, count: number): Promise<void> {
    await this.shiftCells(sheetId, 'row', startRow + count, -count);
  }

  async shiftCellsRight(sheetId: string, startCol: number, count: number): Promise<void> {
    await this.shiftCells(sheetId, 'col', startCol, count);
  }

  async shiftCellsLeft(sheetId: string, startCol: number, count: number): Promise<void> {
    await this.shiftCells(sheetId, 'col', startCol + count, -count);
  }

  private async shiftCells(
    sheetId: string,
    axis: 'row' | 'col',
    startIndex: number,
    delta: number
  ): Promise<void> {
    const rows = await this.sql<Array<{
      tile_row: number;
      tile_col: number;
      values_blob: string;
    }>>`
      SELECT tile_row, tile_col, values_blob FROM sheet_tiles WHERE sheet_id = ${sheetId}
    `;

    // Collect all cells with their absolute positions
    const allCells: Array<{
      row: number;
      col: number;
      cell: TileCell;
    }> = [];

    for (const row of rows) {
      const tile = deserializeTile(row.values_blob);
      for (const [key, tc] of Object.entries(tile.cells)) {
        if (!tc) continue;
        const parsed = parseTileCellKey(key);
        if (!parsed) continue;
        const [offsetRow, offsetCol] = parsed;
        const [cellRow, cellCol] = tileToCell(row.tile_row, row.tile_col, offsetRow, offsetCol);
        allCells.push({ row: cellRow, col: cellCol, cell: tc });
      }
    }

    // Clear all tiles
    await this.sql`DELETE FROM sheet_tiles WHERE sheet_id = ${sheetId}`;

    // Group shifted cells by new tile
    const newTiles = new Map<string, Tile>();

    for (const { row, col, cell } of allCells) {
      let newRow = row;
      let newCol = col;

      if (axis === 'row' && row >= startIndex) {
        newRow = row + delta;
        if (newRow < 0) continue; // Cell shifted off grid
      } else if (axis === 'col' && col >= startIndex) {
        newCol = col + delta;
        if (newCol < 0) continue; // Cell shifted off grid
      }

      const pos = cellToTile(newRow, newCol);
      const tileKey = `${pos.tileRow}:${pos.tileCol}`;

      if (!newTiles.has(tileKey)) {
        newTiles.set(tileKey, createEmptyTile());
      }

      const key = tileCellKey(pos.offsetRow, pos.offsetCol);
      newTiles.get(tileKey)!.cells[key] = cell;
    }

    // Save all new tiles
    for (const [tileKey, tile] of newTiles) {
      const [tileRowStr, tileColStr] = tileKey.split(':');
      const tileRow = parseInt(tileRowStr, 10);
      const tileCol = parseInt(tileColStr, 10);
      await this.saveTile(sheetId, tileRow, tileCol, tile);
    }
  }

  // ============================================================================
  // Merged Regions
  // ============================================================================

  async getMergedRegions(sheetId: string): Promise<MergedRegion[]> {
    return this.sql<MergedRegion[]>`
      SELECT * FROM merged_regions WHERE sheet_id = ${sheetId}
    `;
  }

  async createMergedRegion(input: CreateMergeInput & { id: string; sheet_id: string }): Promise<MergedRegion> {
    const [region] = await this.sql<[MergedRegion]>`
      INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
      VALUES (${input.id}, ${input.sheet_id}, ${input.start_row}, ${input.start_col}, ${input.end_row}, ${input.end_col})
      RETURNING *
    `;
    return region;
  }

  async deleteMergedRegion(id: string): Promise<void> {
    await this.sql`DELETE FROM merged_regions WHERE id = ${id}`;
  }

  async deleteMergedRegionsInRange(
    sheetId: string,
    startRow: number,
    endRow: number,
    startCol: number,
    endCol: number
  ): Promise<void> {
    await this.sql`
      DELETE FROM merged_regions
      WHERE sheet_id = ${sheetId}
        AND start_row >= ${startRow} AND end_row <= ${endRow}
        AND start_col >= ${startCol} AND end_col <= ${endCol}
    `;
  }
}

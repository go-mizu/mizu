/**
 * Tile utilities for tile-based cell storage
 *
 * Tiles are fixed-size blocks of the spreadsheet grid (256 rows x 64 columns).
 * This matches the Go store implementation for consistency.
 */

export const TILE_HEIGHT = 256;
export const TILE_WIDTH = 64;

/**
 * Cell data stored within a tile
 */
export interface TileCell {
  id?: string;
  value?: string | null;
  formula?: string | null;
  display?: string | null;
  format?: string | null;
}

/**
 * A tile containing multiple cells
 * Keys are "offsetRow,offsetCol" within the tile
 */
export interface Tile {
  cells: Record<string, TileCell>;
}

/**
 * Result of cellToTile conversion
 */
export interface TilePosition {
  tileRow: number;
  tileCol: number;
  offsetRow: number;
  offsetCol: number;
}

/**
 * Convert cell position to tile coordinates
 *
 * @param row - Cell row (0-indexed)
 * @param col - Cell column (0-indexed)
 * @returns Tile coordinates and offset within tile
 */
export function cellToTile(row: number, col: number): TilePosition {
  return {
    tileRow: Math.floor(row / TILE_HEIGHT),
    tileCol: Math.floor(col / TILE_WIDTH),
    offsetRow: row % TILE_HEIGHT,
    offsetCol: col % TILE_WIDTH,
  };
}

/**
 * Create a key for a cell within a tile
 *
 * @param offsetRow - Row offset within tile (0-255)
 * @param offsetCol - Column offset within tile (0-63)
 * @returns Key string "offsetRow,offsetCol"
 */
export function tileCellKey(offsetRow: number, offsetCol: number): string {
  return `${offsetRow},${offsetCol}`;
}

/**
 * Parse a tile cell key back to offsets
 *
 * @param key - Key string "offsetRow,offsetCol"
 * @returns [offsetRow, offsetCol] or null if invalid
 */
export function parseTileCellKey(key: string): [number, number] | null {
  const parts = key.split(',');
  if (parts.length !== 2) return null;
  const offsetRow = parseInt(parts[0], 10);
  const offsetCol = parseInt(parts[1], 10);
  if (isNaN(offsetRow) || isNaN(offsetCol)) return null;
  return [offsetRow, offsetCol];
}

/**
 * Convert tile coordinates back to cell position
 *
 * @param tileRow - Tile row
 * @param tileCol - Tile column
 * @param offsetRow - Row offset within tile
 * @param offsetCol - Column offset within tile
 * @returns [row, col] cell position
 */
export function tileToCell(
  tileRow: number,
  tileCol: number,
  offsetRow: number,
  offsetCol: number
): [number, number] {
  return [
    tileRow * TILE_HEIGHT + offsetRow,
    tileCol * TILE_WIDTH + offsetCol,
  ];
}

/**
 * Calculate tile range for a cell range
 *
 * @param startRow - Start row of cell range
 * @param startCol - Start column of cell range
 * @param endRow - End row of cell range
 * @param endCol - End column of cell range
 * @returns Tile range bounds
 */
export function getTileRange(
  startRow: number,
  startCol: number,
  endRow: number,
  endCol: number
): {
  startTileRow: number;
  startTileCol: number;
  endTileRow: number;
  endTileCol: number;
} {
  return {
    startTileRow: Math.floor(startRow / TILE_HEIGHT),
    startTileCol: Math.floor(startCol / TILE_WIDTH),
    endTileRow: Math.floor(endRow / TILE_HEIGHT),
    endTileCol: Math.floor(endCol / TILE_WIDTH),
  };
}

/**
 * Create an empty tile
 */
export function createEmptyTile(): Tile {
  return { cells: {} };
}

/**
 * Serialize a tile to JSON blob
 */
export function serializeTile(tile: Tile): string {
  return JSON.stringify(tile);
}

/**
 * Deserialize a tile from JSON blob
 */
export function deserializeTile(blob: string | null | undefined): Tile {
  if (!blob) return createEmptyTile();
  try {
    const parsed = JSON.parse(blob);
    return {
      cells: parsed.cells || {},
    };
  } catch {
    return createEmptyTile();
  }
}

/**
 * Check if a tile is empty (no cells)
 */
export function isTileEmpty(tile: Tile): boolean {
  return Object.keys(tile.cells).length === 0;
}

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/pkg/formula"
)

// Tile dimensions
const (
	TileHeight = 256
	TileWidth  = 64
)

// TileCell represents a cell within a tile.
type TileCell struct {
	ID        string          `json:"id,omitempty"`
	Value     interface{}     `json:"value,omitempty"`
	Formula   string          `json:"formula,omitempty"`
	Display   string          `json:"display,omitempty"`
	Type      cells.CellType  `json:"type,omitempty"`
	Format    *cells.Format   `json:"format,omitempty"`
	Hyperlink *cells.Hyperlink `json:"hyperlink,omitempty"`
	Note      string          `json:"note,omitempty"`
}

// Tile represents a tile of cells.
type Tile struct {
	Cells map[string]*TileCell `json:"cells"` // key: "row,col" within tile
}

// CellsStore implements cells.Store with tile-based storage.
type CellsStore struct {
	db *sql.DB
}

// NewCellsStore creates a new cells store.
func NewCellsStore(db *sql.DB) *CellsStore {
	return &CellsStore{db: db}
}

// cellToTile converts cell position to tile coordinates.
func cellToTile(row, col int) (tileRow, tileCol, offsetRow, offsetCol int) {
	tileRow = row / TileHeight
	tileCol = col / TileWidth
	offsetRow = row % TileHeight
	offsetCol = col % TileWidth
	return
}

// tileCellKey creates a key for a cell within a tile.
func tileCellKey(offsetRow, offsetCol int) string {
	return string(rune(offsetRow)) + "," + string(rune(offsetCol))
}

// loadTile loads a tile from the database.
func (s *CellsStore) loadTile(ctx context.Context, sheetID string, tileRow, tileCol int) (*Tile, error) {
	var blob []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT values_blob FROM sheet_tiles
		WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
	`, sheetID, tileRow, tileCol).Scan(&blob)

	if err == sql.ErrNoRows {
		return &Tile{Cells: make(map[string]*TileCell)}, nil
	}
	if err != nil {
		return nil, err
	}

	tile := &Tile{Cells: make(map[string]*TileCell)}
	if len(blob) > 0 {
		json.Unmarshal(blob, tile)
	}
	return tile, nil
}

// saveTile saves a tile to the database.
func (s *CellsStore) saveTile(ctx context.Context, sheetID string, tileRow, tileCol int, tile *Tile) error {
	blob, _ := json.Marshal(tile)

	// If tile is empty, delete it
	if len(tile.Cells) == 0 {
		_, err := s.db.ExecContext(ctx, `
			DELETE FROM sheet_tiles WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
		`, sheetID, tileRow, tileCol)
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sheet_tiles (sheet_id, tile_row, tile_col, tile_h, tile_w, encoding, values_blob, updated_at)
		VALUES (?, ?, ?, ?, ?, 'json_v1', ?, CURRENT_TIMESTAMP)
		ON CONFLICT (sheet_id, tile_row, tile_col) DO UPDATE SET
			values_blob = excluded.values_blob,
			updated_at = excluded.updated_at
	`, sheetID, tileRow, tileCol, TileHeight, TileWidth, blob)
	return err
}

// Get retrieves a cell by position.
func (s *CellsStore) Get(ctx context.Context, sheetID string, row, col int) (*cells.Cell, error) {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(row, col)
	tile, err := s.loadTile(ctx, sheetID, tileRow, tileCol)
	if err != nil {
		return nil, err
	}

	key := tileCellKey(offsetRow, offsetCol)
	tc, ok := tile.Cells[key]
	if !ok || tc == nil {
		return nil, cells.ErrNotFound
	}

	cell := &cells.Cell{
		ID:        tc.ID,
		SheetID:   sheetID,
		Row:       row,
		Col:       col,
		Value:     tc.Value,
		Formula:   tc.Formula,
		Display:   tc.Display,
		Type:      tc.Type,
		Hyperlink: tc.Hyperlink,
		Note:      tc.Note,
	}
	if tc.Format != nil {
		cell.Format = *tc.Format
	}

	return cell, nil
}

// GetRange retrieves cells in a range.
func (s *CellsStore) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	// Calculate tile range
	startTileRow := startRow / TileHeight
	startTileCol := startCol / TileWidth
	endTileRow := endRow / TileHeight
	endTileCol := endCol / TileWidth

	result := make([]*cells.Cell, 0)

	// Load all relevant tiles
	rows, err := s.db.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles
		WHERE sheet_id = ?
			AND tile_row >= ? AND tile_row <= ?
			AND tile_col >= ? AND tile_col <= ?
	`, sheetID, startTileRow, endTileRow, startTileCol, endTileCol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tileRow, tileCol int
		var blob []byte
		if err := rows.Scan(&tileRow, &tileCol, &blob); err != nil {
			return nil, err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		// Extract cells within the requested range
		for key, tc := range tile.Cells {
			if tc == nil {
				continue
			}
			var offsetRow, offsetCol int
			// Parse key
			if len(key) >= 3 {
				offsetRow = int(key[0])
				offsetCol = int(key[2])
			}

			cellRow := tileRow*TileHeight + offsetRow
			cellCol := tileCol*TileWidth + offsetCol

			if cellRow >= startRow && cellRow <= endRow && cellCol >= startCol && cellCol <= endCol {
				cell := &cells.Cell{
					ID:        tc.ID,
					SheetID:   sheetID,
					Row:       cellRow,
					Col:       cellCol,
					Value:     tc.Value,
					Formula:   tc.Formula,
					Display:   tc.Display,
					Type:      tc.Type,
					Hyperlink: tc.Hyperlink,
					Note:      tc.Note,
				}
				if tc.Format != nil {
					cell.Format = *tc.Format
				}
				result = append(result, cell)
			}
		}
	}

	// Sort by row, then col
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Row > result[j].Row ||
				(result[i].Row == result[j].Row && result[i].Col > result[j].Col) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// Set sets a cell.
func (s *CellsStore) Set(ctx context.Context, cell *cells.Cell) error {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
	tile, err := s.loadTile(ctx, cell.SheetID, tileRow, tileCol)
	if err != nil {
		return err
	}

	key := tileCellKey(offsetRow, offsetCol)
	tc := &TileCell{
		ID:        cell.ID,
		Value:     cell.Value,
		Formula:   cell.Formula,
		Display:   cell.Display,
		Type:      cell.Type,
		Format:    &cell.Format,
		Hyperlink: cell.Hyperlink,
		Note:      cell.Note,
	}
	tile.Cells[key] = tc

	return s.saveTile(ctx, cell.SheetID, tileRow, tileCol, tile)
}

// BatchSet sets multiple cells.
func (s *CellsStore) BatchSet(ctx context.Context, cellList []*cells.Cell) error {
	if len(cellList) == 0 {
		return nil
	}

	// Group cells by tile
	type tileKey struct {
		sheetID string
		tileRow int
		tileCol int
	}
	tileUpdates := make(map[tileKey]map[string]*TileCell)

	for _, cell := range cellList {
		tileRow, tileCol, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
		key := tileKey{cell.SheetID, tileRow, tileCol}

		if _, ok := tileUpdates[key]; !ok {
			tileUpdates[key] = make(map[string]*TileCell)
		}

		cellKey := tileCellKey(offsetRow, offsetCol)
		tileUpdates[key][cellKey] = &TileCell{
			ID:        cell.ID,
			Value:     cell.Value,
			Formula:   cell.Formula,
			Display:   cell.Display,
			Type:      cell.Type,
			Format:    &cell.Format,
			Hyperlink: cell.Hyperlink,
			Note:      cell.Note,
		}
	}

	// Update each tile
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for key, updates := range tileUpdates {
		// Load existing tile
		var blob []byte
		err := tx.QueryRowContext(ctx, `
			SELECT values_blob FROM sheet_tiles
			WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
		`, key.sheetID, key.tileRow, key.tileCol).Scan(&blob)

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if err == nil && len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		// Apply updates
		for cellKey, tc := range updates {
			tile.Cells[cellKey] = tc
		}

		// Save tile
		newBlob, _ := json.Marshal(tile)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO sheet_tiles (sheet_id, tile_row, tile_col, tile_h, tile_w, encoding, values_blob, updated_at)
			VALUES (?, ?, ?, ?, ?, 'json_v1', ?, CURRENT_TIMESTAMP)
			ON CONFLICT (sheet_id, tile_row, tile_col) DO UPDATE SET
				values_blob = excluded.values_blob,
				updated_at = excluded.updated_at
		`, key.sheetID, key.tileRow, key.tileCol, TileHeight, TileWidth, newBlob)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete deletes a cell.
func (s *CellsStore) Delete(ctx context.Context, sheetID string, row, col int) error {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(row, col)
	tile, err := s.loadTile(ctx, sheetID, tileRow, tileCol)
	if err != nil {
		return err
	}

	key := tileCellKey(offsetRow, offsetCol)
	delete(tile.Cells, key)

	return s.saveTile(ctx, sheetID, tileRow, tileCol, tile)
}

// DeleteRange deletes cells in a range.
func (s *CellsStore) DeleteRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	startTileRow := startRow / TileHeight
	startTileCol := startCol / TileWidth
	endTileRow := endRow / TileHeight
	endTileCol := endCol / TileWidth

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles
		WHERE sheet_id = ?
			AND tile_row >= ? AND tile_row <= ?
			AND tile_col >= ? AND tile_col <= ?
	`, sheetID, startTileRow, endTileRow, startTileCol, endTileCol)
	if err != nil {
		return err
	}

	type tileData struct {
		tileRow int
		tileCol int
		tile    *Tile
	}
	tilesToUpdate := make([]tileData, 0)

	for rows.Next() {
		var tr, tc int
		var blob []byte
		if err := rows.Scan(&tr, &tc, &blob); err != nil {
			rows.Close()
			return err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		// Delete cells in range
		for key := range tile.Cells {
			var offsetRow, offsetCol int
			if len(key) >= 3 {
				offsetRow = int(key[0])
				offsetCol = int(key[2])
			}

			cellRow := tr*TileHeight + offsetRow
			cellCol := tc*TileWidth + offsetCol

			if cellRow >= startRow && cellRow <= endRow && cellCol >= startCol && cellCol <= endCol {
				delete(tile.Cells, key)
			}
		}

		tilesToUpdate = append(tilesToUpdate, tileData{tr, tc, tile})
	}
	rows.Close()

	// Update modified tiles
	for _, td := range tilesToUpdate {
		if len(td.tile.Cells) == 0 {
			_, err = tx.ExecContext(ctx, `
				DELETE FROM sheet_tiles WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, sheetID, td.tileRow, td.tileCol)
		} else {
			blob, _ := json.Marshal(td.tile)
			_, err = tx.ExecContext(ctx, `
				UPDATE sheet_tiles SET values_blob = ?, updated_at = CURRENT_TIMESTAMP
				WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, blob, sheetID, td.tileRow, td.tileCol)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CreateMerge creates a merged region.
func (s *CellsStore) CreateMerge(ctx context.Context, region *cells.MergedRegion) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
		VALUES (?, ?, ?, ?, ?, ?)
	`, region.ID, region.SheetID, region.StartRow, region.StartCol, region.EndRow, region.EndCol)
	return err
}

// BatchCreateMerge creates multiple merged regions.
func (s *CellsStore) BatchCreateMerge(ctx context.Context, regions []*cells.MergedRegion) error {
	if len(regions) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, region := range regions {
		if _, err := stmt.ExecContext(ctx, region.ID, region.SheetID,
			region.StartRow, region.StartCol, region.EndRow, region.EndCol); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteMerge deletes a merged region.
func (s *CellsStore) DeleteMerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM merged_regions
		WHERE sheet_id = ? AND start_row = ? AND start_col = ? AND end_row = ? AND end_col = ?
	`, sheetID, startRow, startCol, endRow, endCol)
	return err
}

// GetMergedRegions retrieves merged regions for a sheet.
func (s *CellsStore) GetMergedRegions(ctx context.Context, sheetID string) ([]*cells.MergedRegion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sheet_id, start_row, start_col, end_row, end_col
		FROM merged_regions WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*cells.MergedRegion, 0)
	for rows.Next() {
		region := &cells.MergedRegion{}
		if err := rows.Scan(&region.ID, &region.SheetID, &region.StartRow,
			&region.StartCol, &region.EndRow, &region.EndCol); err != nil {
			return nil, err
		}
		result = append(result, region)
	}
	return result, nil
}

// ShiftRows shifts rows (for insert/delete operations).
func (s *CellsStore) ShiftRows(ctx context.Context, sheetID string, startRow, count int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update formulas first
	if err := s.shiftFormulasInTx(ctx, tx, sheetID, "row", startRow, count); err != nil {
		return err
	}

	// Load all tiles and shift cells
	rows, err := tx.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return err
	}

	type tileUpdate struct {
		tileRow int
		tileCol int
		tile    *Tile
	}
	updates := make([]tileUpdate, 0)

	for rows.Next() {
		var tr, tc int
		var blob []byte
		if err := rows.Scan(&tr, &tc, &blob); err != nil {
			rows.Close()
			return err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		// Collect cells to shift
		newCells := make(map[string]*TileCell)
		for key, cell := range tile.Cells {
			var offsetRow, offsetCol int
			if len(key) >= 3 {
				offsetRow = int(key[0])
				offsetCol = int(key[2])
			}

			cellRow := tr*TileHeight + offsetRow

			if count > 0 {
				// Insert - shift down
				if cellRow >= startRow {
					newRow := cellRow + count
					newTileRow := newRow / TileHeight
					newOffsetRow := newRow % TileHeight
					if newTileRow == tr {
						newKey := tileCellKey(newOffsetRow, offsetCol)
						newCells[newKey] = cell
					}
					// If moved to different tile, handle separately (simplified for now)
				} else {
					newCells[key] = cell
				}
			} else {
				// Delete
				if cellRow == startRow {
					// This cell is deleted
					continue
				} else if cellRow > startRow {
					newRow := cellRow - 1
					newTileRow := newRow / TileHeight
					newOffsetRow := newRow % TileHeight
					if newTileRow == tr {
						newKey := tileCellKey(newOffsetRow, offsetCol)
						newCells[newKey] = cell
					}
				} else {
					newCells[key] = cell
				}
			}
		}

		tile.Cells = newCells
		updates = append(updates, tileUpdate{tr, tc, tile})
	}
	rows.Close()

	// Save updated tiles
	for _, u := range updates {
		if len(u.tile.Cells) == 0 {
			_, err = tx.ExecContext(ctx, `
				DELETE FROM sheet_tiles WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, sheetID, u.tileRow, u.tileCol)
		} else {
			blob, _ := json.Marshal(u.tile)
			_, err = tx.ExecContext(ctx, `
				UPDATE sheet_tiles SET values_blob = ?, updated_at = CURRENT_TIMESTAMP
				WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, blob, sheetID, u.tileRow, u.tileCol)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ShiftCols shifts columns (for insert/delete operations).
func (s *CellsStore) ShiftCols(ctx context.Context, sheetID string, startCol, count int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update formulas first
	if err := s.shiftFormulasInTx(ctx, tx, sheetID, "col", startCol, count); err != nil {
		return err
	}

	// Load all tiles and shift cells
	rows, err := tx.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return err
	}

	type tileUpdate struct {
		tileRow int
		tileCol int
		tile    *Tile
	}
	updates := make([]tileUpdate, 0)

	for rows.Next() {
		var tr, tc int
		var blob []byte
		if err := rows.Scan(&tr, &tc, &blob); err != nil {
			rows.Close()
			return err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		// Collect cells to shift
		newCells := make(map[string]*TileCell)
		for key, cell := range tile.Cells {
			var offsetRow, offsetCol int
			if len(key) >= 3 {
				offsetRow = int(key[0])
				offsetCol = int(key[2])
			}

			cellCol := tc*TileWidth + offsetCol

			if count > 0 {
				// Insert - shift right
				if cellCol >= startCol {
					newCol := cellCol + count
					newTileCol := newCol / TileWidth
					newOffsetCol := newCol % TileWidth
					if newTileCol == tc {
						newKey := tileCellKey(offsetRow, newOffsetCol)
						newCells[newKey] = cell
					}
				} else {
					newCells[key] = cell
				}
			} else {
				// Delete
				if cellCol == startCol {
					continue
				} else if cellCol > startCol {
					newCol := cellCol - 1
					newTileCol := newCol / TileWidth
					newOffsetCol := newCol % TileWidth
					if newTileCol == tc {
						newKey := tileCellKey(offsetRow, newOffsetCol)
						newCells[newKey] = cell
					}
				} else {
					newCells[key] = cell
				}
			}
		}

		tile.Cells = newCells
		updates = append(updates, tileUpdate{tr, tc, tile})
	}
	rows.Close()

	// Save updated tiles
	for _, u := range updates {
		if len(u.tile.Cells) == 0 {
			_, err = tx.ExecContext(ctx, `
				DELETE FROM sheet_tiles WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, sheetID, u.tileRow, u.tileCol)
		} else {
			blob, _ := json.Marshal(u.tile)
			_, err = tx.ExecContext(ctx, `
				UPDATE sheet_tiles SET values_blob = ?, updated_at = CURRENT_TIMESTAMP
				WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, blob, sheetID, u.tileRow, u.tileCol)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByPositions retrieves multiple cells by their positions.
func (s *CellsStore) GetByPositions(ctx context.Context, sheetID string, positions []cells.CellPosition) (map[cells.CellPosition]*cells.Cell, error) {
	if len(positions) == 0 {
		return make(map[cells.CellPosition]*cells.Cell), nil
	}

	result := make(map[cells.CellPosition]*cells.Cell, len(positions))

	// Group positions by tile
	type tileKey struct {
		tileRow int
		tileCol int
	}
	tilePositions := make(map[tileKey][]cells.CellPosition)

	for _, pos := range positions {
		tileRow, tileCol, _, _ := cellToTile(pos.Row, pos.Col)
		key := tileKey{tileRow, tileCol}
		tilePositions[key] = append(tilePositions[key], pos)
	}

	// Load relevant tiles
	for key, posInTile := range tilePositions {
		tile, err := s.loadTile(ctx, sheetID, key.tileRow, key.tileCol)
		if err != nil {
			return nil, err
		}

		for _, pos := range posInTile {
			_, _, offsetRow, offsetCol := cellToTile(pos.Row, pos.Col)
			cellKey := tileCellKey(offsetRow, offsetCol)

			if tc, ok := tile.Cells[cellKey]; ok && tc != nil {
				cell := &cells.Cell{
					ID:        tc.ID,
					SheetID:   sheetID,
					Row:       pos.Row,
					Col:       pos.Col,
					Value:     tc.Value,
					Formula:   tc.Formula,
					Display:   tc.Display,
					Type:      tc.Type,
					Hyperlink: tc.Hyperlink,
					Note:      tc.Note,
				}
				if tc.Format != nil {
					cell.Format = *tc.Format
				}
				result[pos] = cell
			}
		}
	}

	return result, nil
}

// DeleteRowsRange deletes multiple rows.
func (s *CellsStore) DeleteRowsRange(ctx context.Context, sheetID string, startRow, count int) error {
	if count <= 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update formulas
	if err := s.shiftFormulasInTx(ctx, tx, sheetID, "row", startRow, -count); err != nil {
		return err
	}

	// Load all tiles
	rows, err := tx.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return err
	}

	type tileUpdate struct {
		tileRow int
		tileCol int
		tile    *Tile
	}
	updates := make([]tileUpdate, 0)
	endRow := startRow + count - 1

	for rows.Next() {
		var tr, tc int
		var blob []byte
		if err := rows.Scan(&tr, &tc, &blob); err != nil {
			rows.Close()
			return err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		newCells := make(map[string]*TileCell)
		for key, cell := range tile.Cells {
			var offsetRow, offsetCol int
			if len(key) >= 3 {
				offsetRow = int(key[0])
				offsetCol = int(key[2])
			}

			cellRow := tr*TileHeight + offsetRow

			if cellRow >= startRow && cellRow <= endRow {
				// Delete this cell
				continue
			} else if cellRow > endRow {
				// Shift up
				newRow := cellRow - count
				newTileRow := newRow / TileHeight
				newOffsetRow := newRow % TileHeight
				if newTileRow == tr {
					newKey := tileCellKey(newOffsetRow, offsetCol)
					newCells[newKey] = cell
				}
			} else {
				newCells[key] = cell
			}
		}

		tile.Cells = newCells
		updates = append(updates, tileUpdate{tr, tc, tile})
	}
	rows.Close()

	for _, u := range updates {
		if len(u.tile.Cells) == 0 {
			_, err = tx.ExecContext(ctx, `
				DELETE FROM sheet_tiles WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, sheetID, u.tileRow, u.tileCol)
		} else {
			blob, _ := json.Marshal(u.tile)
			_, err = tx.ExecContext(ctx, `
				UPDATE sheet_tiles SET values_blob = ?, updated_at = CURRENT_TIMESTAMP
				WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, blob, sheetID, u.tileRow, u.tileCol)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteColsRange deletes multiple columns.
func (s *CellsStore) DeleteColsRange(ctx context.Context, sheetID string, startCol, count int) error {
	if count <= 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update formulas
	if err := s.shiftFormulasInTx(ctx, tx, sheetID, "col", startCol, -count); err != nil {
		return err
	}

	// Load all tiles
	rows, err := tx.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return err
	}

	type tileUpdate struct {
		tileRow int
		tileCol int
		tile    *Tile
	}
	updates := make([]tileUpdate, 0)
	endCol := startCol + count - 1

	for rows.Next() {
		var tr, tc int
		var blob []byte
		if err := rows.Scan(&tr, &tc, &blob); err != nil {
			rows.Close()
			return err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		newCells := make(map[string]*TileCell)
		for key, cell := range tile.Cells {
			var offsetRow, offsetCol int
			if len(key) >= 3 {
				offsetRow = int(key[0])
				offsetCol = int(key[2])
			}

			cellCol := tc*TileWidth + offsetCol

			if cellCol >= startCol && cellCol <= endCol {
				continue
			} else if cellCol > endCol {
				newCol := cellCol - count
				newTileCol := newCol / TileWidth
				newOffsetCol := newCol % TileWidth
				if newTileCol == tc {
					newKey := tileCellKey(offsetRow, newOffsetCol)
					newCells[newKey] = cell
				}
			} else {
				newCells[key] = cell
			}
		}

		tile.Cells = newCells
		updates = append(updates, tileUpdate{tr, tc, tile})
	}
	rows.Close()

	for _, u := range updates {
		if len(u.tile.Cells) == 0 {
			_, err = tx.ExecContext(ctx, `
				DELETE FROM sheet_tiles WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, sheetID, u.tileRow, u.tileCol)
		} else {
			blob, _ := json.Marshal(u.tile)
			_, err = tx.ExecContext(ctx, `
				UPDATE sheet_tiles SET values_blob = ?, updated_at = CURRENT_TIMESTAMP
				WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
			`, blob, sheetID, u.tileRow, u.tileCol)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// shiftFormulasInTx updates formula references.
func (s *CellsStore) shiftFormulasInTx(ctx context.Context, tx *sql.Tx, sheetID, shiftType string, startIndex, count int) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT tile_row, tile_col, values_blob FROM sheet_tiles WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return err
	}

	type formulaUpdate struct {
		tileRow int
		tileCol int
		key     string
		formula string
	}
	updates := make([]formulaUpdate, 0)

	for rows.Next() {
		var tr, tc int
		var blob []byte
		if err := rows.Scan(&tr, &tc, &blob); err != nil {
			rows.Close()
			return err
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		for key, cell := range tile.Cells {
			if cell != nil && cell.Formula != "" {
				newFormula := formula.ShiftFormula(cell.Formula, shiftType, startIndex, count, "")
				if newFormula != cell.Formula {
					updates = append(updates, formulaUpdate{tr, tc, key, newFormula})
				}
			}
		}
	}
	rows.Close()

	// Group updates by tile
	tileUpdates := make(map[string]map[string]string)
	for _, u := range updates {
		key := string(rune(u.tileRow)) + "," + string(rune(u.tileCol))
		if _, ok := tileUpdates[key]; !ok {
			tileUpdates[key] = make(map[string]string)
		}
		tileUpdates[key][u.key] = u.formula
	}

	// Apply updates
	for tileKey, cellUpdates := range tileUpdates {
		var tr, tc int
		if len(tileKey) >= 3 {
			tr = int(tileKey[0])
			tc = int(tileKey[2])
		}

		var blob []byte
		err := tx.QueryRowContext(ctx, `
			SELECT values_blob FROM sheet_tiles
			WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
		`, sheetID, tr, tc).Scan(&blob)
		if err != nil {
			continue
		}

		tile := &Tile{Cells: make(map[string]*TileCell)}
		if len(blob) > 0 {
			json.Unmarshal(blob, tile)
		}

		for cellKey, newFormula := range cellUpdates {
			if cell, ok := tile.Cells[cellKey]; ok && cell != nil {
				cell.Formula = newFormula
			}
		}

		newBlob, _ := json.Marshal(tile)
		_, err = tx.ExecContext(ctx, `
			UPDATE sheet_tiles SET values_blob = ?, updated_at = CURRENT_TIMESTAMP
			WHERE sheet_id = ? AND tile_row = ? AND tile_col = ?
		`, newBlob, sheetID, tr, tc)
		if err != nil {
			return err
		}
	}

	return nil
}

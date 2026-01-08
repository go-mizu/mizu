package cached

import (
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
)

// Tile represents a tile of cells.
type Tile struct {
	Cells map[string]*TileCell // key: "offsetRow,offsetCol" within tile
}

// TileCell represents a cell within a tile.
type TileCell struct {
	ID        string
	Value     interface{}
	Formula   string
	Display   string
	Type      cells.CellType
	Format    *cells.Format
	Hyperlink *cells.Hyperlink
	Note      string
	UpdatedAt time.Time
}

// cellToTileCell converts a Cell to a TileCell.
func cellToTileCell(cell *cells.Cell) *TileCell {
	tc := &TileCell{
		ID:        cell.ID,
		Value:     cell.Value,
		Formula:   cell.Formula,
		Display:   cell.Display,
		Type:      cell.Type,
		Hyperlink: cell.Hyperlink,
		Note:      cell.Note,
		UpdatedAt: cell.UpdatedAt,
	}

	// Copy format if non-zero
	if cell.Format != (cells.Format{}) {
		f := cell.Format
		tc.Format = &f
	}

	return tc
}

// tileCellToCell converts a TileCell back to a Cell.
func tileCellToCell(tc *TileCell, sheetID string, row, col int) *cells.Cell {
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
		UpdatedAt: tc.UpdatedAt,
	}

	if tc.Format != nil {
		cell.Format = *tc.Format
	}

	return cell
}

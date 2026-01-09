// Package cached provides a high-performance in-memory cache wrapper for cells.Store.
package cached

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
)

// Tile dimensions matching underlying stores
const (
	TileHeight = 256
	TileWidth  = 64
)

// Config holds cache configuration.
type Config struct {
	// FlushInterval for periodic background flush
	FlushInterval time.Duration

	// FlushThreshold triggers flush when dirty tile count exceeds
	FlushThreshold int
}

// DefaultConfig returns sensible defaults.
var DefaultConfig = Config{
	FlushInterval:  time.Second,
	FlushThreshold: 100,
}

// tileKey identifies a tile uniquely.
type tileKey struct {
	sheetID string
	tileRow int
	tileCol int
}

// cellPos is a compact cell position for map keys.
type cellPos struct {
	row, col int
}

// sheetCache holds all tiles for a single sheet.
type sheetCache struct {
	mu          sync.RWMutex
	tiles       map[tileKey]*Tile
	loadedTiles map[tileKey]bool // tracks which tiles have been loaded from underlying (for optimized mode)
	fullyLoaded bool             // tracks if entire sheet has been loaded (for optimized mode)

	// Optimized indexes (only populated in optimized mode after full load)
	cellIndex map[cellPos]*TileCell // direct position -> cell lookup
	rowIndex  map[int][]int         // row -> sorted list of columns with data
	minRow    int                   // minimum row with data
	maxRow    int                   // maximum row with data
	minCol    int                   // minimum column with data
	maxCol    int                   // maximum column with data
}

// Store implements cells.Store with full in-memory caching.
type Store struct {
	underlying cells.Store
	config     Config

	// Per-sheet tile caches (sheetID -> *sheetCache)
	sheets sync.Map

	// Merge region cache (sheetID -> []*MergedRegion)
	merges sync.Map

	// Dirty tracking
	mu          sync.RWMutex
	dirtyTiles  map[tileKey]struct{}
	dirtyMerges map[string]struct{}
	deletedTiles map[tileKey]struct{}

	// Background flusher
	flushTicker *time.Ticker
	stopCh      chan struct{}
	wg          sync.WaitGroup

	// Stats
	stats Stats
}

// Stats holds cache statistics.
type Stats struct {
	HitCount   int64
	MissCount  int64
	FlushCount int64
}

// New creates a new cached store wrapping the underlying store.
func New(underlying cells.Store, config Config) *Store {
	if config.FlushInterval == 0 {
		config.FlushInterval = DefaultConfig.FlushInterval
	}
	if config.FlushThreshold == 0 {
		config.FlushThreshold = DefaultConfig.FlushThreshold
	}

	s := &Store{
		underlying:   underlying,
		config:       config,
		dirtyTiles:   make(map[tileKey]struct{}),
		dirtyMerges:  make(map[string]struct{}),
		deletedTiles: make(map[tileKey]struct{}),
		stopCh:       make(chan struct{}),
	}

	s.startBackgroundFlusher()
	return s
}

// startBackgroundFlusher starts the periodic flush goroutine.
func (s *Store) startBackgroundFlusher() {
	s.flushTicker = time.NewTicker(s.config.FlushInterval)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.flushTicker.C:
				_ = s.Flush(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Close flushes remaining data and stops background workers.
func (s *Store) Close() error {
	// Stop background flusher
	close(s.stopCh)
	s.flushTicker.Stop()
	s.wg.Wait()

	// Final flush
	return s.Flush(context.Background())
}

// getOrCreateSheetCache gets or creates a sheet cache.
func (s *Store) getOrCreateSheetCache(sheetID string) *sheetCache {
	if v, ok := s.sheets.Load(sheetID); ok {
		return v.(*sheetCache)
	}

	sc := &sheetCache{
		tiles:       make(map[tileKey]*Tile),
		loadedTiles: make(map[tileKey]bool),
		cellIndex:   make(map[cellPos]*TileCell),
		rowIndex:    make(map[int][]int),
		minRow:      -1,
		maxRow:      -1,
		minCol:      -1,
		maxCol:      -1,
	}
	actual, _ := s.sheets.LoadOrStore(sheetID, sc)
	return actual.(*sheetCache)
}

// ensureSheetLoaded ensures the sheet is fully loaded into cache (optimized mode only).
// Returns true if the sheet is now fully loaded.
func (s *Store) ensureSheetLoaded(ctx context.Context, sheetID string) (*sheetCache, error) {
	sc := s.getOrCreateSheetCache(sheetID)

	sc.mu.RLock()
	if sc.fullyLoaded {
		sc.mu.RUnlock()
		return sc, nil
	}
	sc.mu.RUnlock()

	// Need to load - upgrade to write lock
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Double-check after acquiring write lock
	if sc.fullyLoaded {
		return sc, nil
	}

	atomic.AddInt64(&s.stats.MissCount, 1)

	// Load ALL cells for this sheet in one query
	// Using a very large range to get everything
	cellList, err := s.underlying.GetRange(ctx, sheetID, 0, 0, 1000000, 10000)
	if err != nil {
		return nil, err
	}

	// Build tiles and indexes
	for _, cell := range cellList {
		tileRow, tileCol, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
		tKey := tileKey{cell.SheetID, tileRow, tileCol}

		// Store in tile
		tile := sc.tiles[tKey]
		if tile == nil {
			tile = &Tile{Cells: make(map[string]*TileCell)}
			sc.tiles[tKey] = tile
		}
		cellKey := tileCellKey(offsetRow, offsetCol)
		tc := cellToTileCell(cell)
		tile.Cells[cellKey] = tc

		// Store in cellIndex
		pos := cellPos{cell.Row, cell.Col}
		sc.cellIndex[pos] = tc

		// Update rowIndex
		sc.rowIndex[cell.Row] = append(sc.rowIndex[cell.Row], cell.Col)

		// Update bounds
		if sc.minRow == -1 || cell.Row < sc.minRow {
			sc.minRow = cell.Row
		}
		if cell.Row > sc.maxRow {
			sc.maxRow = cell.Row
		}
		if sc.minCol == -1 || cell.Col < sc.minCol {
			sc.minCol = cell.Col
		}
		if cell.Col > sc.maxCol {
			sc.maxCol = cell.Col
		}
	}

	// Sort rowIndex columns for binary search
	for row := range sc.rowIndex {
		cols := sc.rowIndex[row]
		sortInts(cols)
	}

	sc.fullyLoaded = true
	return sc, nil
}

// sortInts sorts a slice of ints in place.
func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

// getSheetCache gets a sheet cache if it exists.
func (s *Store) getSheetCache(sheetID string) *sheetCache {
	if v, ok := s.sheets.Load(sheetID); ok {
		return v.(*sheetCache)
	}
	return nil
}

// markDirty marks a tile as dirty.
func (s *Store) markDirty(key tileKey) {
	s.mu.Lock()
	s.dirtyTiles[key] = struct{}{}
	delete(s.deletedTiles, key)
	s.mu.Unlock()

	// Check threshold
	s.mu.RLock()
	count := len(s.dirtyTiles)
	s.mu.RUnlock()

	if count >= s.config.FlushThreshold {
		go s.Flush(context.Background())
	}
}

// markTileDeleted marks a tile for deletion.
func (s *Store) markTileDeleted(key tileKey) {
	s.mu.Lock()
	delete(s.dirtyTiles, key)
	s.deletedTiles[key] = struct{}{}
	s.mu.Unlock()
}

// markMergesDirty marks merges as dirty for a sheet.
func (s *Store) markMergesDirty(sheetID string) {
	s.mu.Lock()
	s.dirtyMerges[sheetID] = struct{}{}
	s.mu.Unlock()
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

// parseTileCellKey parses a tile cell key.
func parseTileCellKey(key string) (offsetRow, offsetCol int) {
	if len(key) >= 3 {
		offsetRow = int(key[0])
		offsetCol = int(key[2])
	}
	return
}

// Get retrieves a cell by position.
func (s *Store) Get(ctx context.Context, sheetID string, row, col int) (*cells.Cell, error) {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(row, col)
	tKey := tileKey{sheetID, tileRow, tileCol}

	sc := s.getSheetCache(sheetID)
	if sc == nil {
		// Cache miss - try underlying store and cache result
		atomic.AddInt64(&s.stats.MissCount, 1)
		cell, err := s.underlying.Get(ctx, sheetID, row, col)
		if err != nil {
			return nil, err
		}
		// Cache this cell
		_ = s.cacheCell(cell)
		return cell, nil
	}

	sc.mu.RLock()
	tile := sc.tiles[tKey]
	sc.mu.RUnlock()

	if tile == nil {
		// Try underlying store
		atomic.AddInt64(&s.stats.MissCount, 1)
		cell, err := s.underlying.Get(ctx, sheetID, row, col)
		if err != nil {
			return nil, err
		}
		_ = s.cacheCell(cell)
		return cell, nil
	}

	key := tileCellKey(offsetRow, offsetCol)
	tc := tile.Cells[key]
	if tc == nil {
		return nil, cells.ErrNotFound
	}

	atomic.AddInt64(&s.stats.HitCount, 1)
	return tileCellToCell(tc, sheetID, row, col), nil
}

// cacheCell caches a single cell without marking dirty.
func (s *Store) cacheCell(cell *cells.Cell) error {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
	tKey := tileKey{cell.SheetID, tileRow, tileCol}

	sc := s.getOrCreateSheetCache(cell.SheetID)

	sc.mu.Lock()
	tile := sc.tiles[tKey]
	if tile == nil {
		tile = &Tile{Cells: make(map[string]*TileCell)}
		sc.tiles[tKey] = tile
	}
	cellKey := tileCellKey(offsetRow, offsetCol)
	tile.Cells[cellKey] = cellToTileCell(cell)
	sc.mu.Unlock()

	return nil
}

// GetRange retrieves cells in a range.
// Always preloads the entire sheet into cache on first access for maximum performance.
func (s *Store) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	// Ensure full sheet is loaded
	sc, err := s.ensureSheetLoaded(ctx, sheetID)
	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&s.stats.HitCount, 1)

	// Use rowIndex for efficient retrieval
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Quick bounds check
	if sc.minRow == -1 || startRow > sc.maxRow || endRow < sc.minRow ||
		startCol > sc.maxCol || endCol < sc.minCol {
		return []*cells.Cell{}, nil
	}

	// Pre-allocate result based on estimated density
	capacity := (endRow - startRow + 1) * (endCol - startCol + 1) / 10
	if capacity < 16 {
		capacity = 16
	}
	result := make([]*cells.Cell, 0, capacity)

	// Iterate through rows in range using rowIndex
	for row := startRow; row <= endRow; row++ {
		cols, ok := sc.rowIndex[row]
		if !ok {
			continue
		}

		// Binary search for start column
		lo := 0
		for lo < len(cols) && cols[lo] < startCol {
			lo++
		}

		// Collect cells in column range
		for i := lo; i < len(cols) && cols[i] <= endCol; i++ {
			col := cols[i]
			pos := cellPos{row, col}
			if tc := sc.cellIndex[pos]; tc != nil {
				result = append(result, tileCellToCell(tc, sheetID, row, col))
			}
		}
	}

	// Results are already sorted by row since we iterate rows in order
	// Just need to sort by column within each row (which is already done via rowIndex)
	return result, nil
}


// sortCells sorts cells by row then column using an efficient algorithm.
func sortCells(cells []*cells.Cell) {
	if len(cells) <= 1 {
		return
	}
	// Quick sort implementation for cells
	quickSortCells(cells, 0, len(cells)-1)
}

func quickSortCells(arr []*cells.Cell, low, high int) {
	if low < high {
		pi := partitionCells(arr, low, high)
		quickSortCells(arr, low, pi-1)
		quickSortCells(arr, pi+1, high)
	}
}

func partitionCells(arr []*cells.Cell, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j].Row < pivot.Row || (arr[j].Row == pivot.Row && arr[j].Col < pivot.Col) {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// GetByPositions retrieves multiple cells by their positions.
// Preloads the entire sheet and uses direct position lookup for O(1) access.
func (s *Store) GetByPositions(ctx context.Context, sheetID string, positions []cells.CellPosition) (map[cells.CellPosition]*cells.Cell, error) {
	if len(positions) == 0 {
		return make(map[cells.CellPosition]*cells.Cell), nil
	}

	// Ensure full sheet is loaded
	sc, err := s.ensureSheetLoaded(ctx, sheetID)
	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&s.stats.HitCount, 1)

	// Direct lookup using cellIndex for O(1) access per position
	result := make(map[cells.CellPosition]*cells.Cell, len(positions))

	sc.mu.RLock()
	for _, pos := range positions {
		key := cellPos{pos.Row, pos.Col}
		if tc := sc.cellIndex[key]; tc != nil {
			result[pos] = tileCellToCell(tc, sheetID, pos.Row, pos.Col)
		}
	}
	sc.mu.RUnlock()

	return result, nil
}

// Set sets a cell.
func (s *Store) Set(ctx context.Context, cell *cells.Cell) error {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
	tKey := tileKey{cell.SheetID, tileRow, tileCol}

	sc := s.getOrCreateSheetCache(cell.SheetID)

	sc.mu.Lock()
	tile := sc.tiles[tKey]
	if tile == nil {
		tile = &Tile{Cells: make(map[string]*TileCell)}
		sc.tiles[tKey] = tile
	}
	cellKey := tileCellKey(offsetRow, offsetCol)
	tc := cellToTileCell(cell)
	tile.Cells[cellKey] = tc

	// Update optimized indexes if sheet is fully loaded
	if sc.fullyLoaded {
		pos := cellPos{cell.Row, cell.Col}
		oldCell := sc.cellIndex[pos]
		sc.cellIndex[pos] = tc

		// Update rowIndex if this is a new cell
		if oldCell == nil {
			sc.rowIndex[cell.Row] = insertSorted(sc.rowIndex[cell.Row], cell.Col)
		}

		// Update bounds
		if sc.minRow == -1 || cell.Row < sc.minRow {
			sc.minRow = cell.Row
		}
		if cell.Row > sc.maxRow {
			sc.maxRow = cell.Row
		}
		if sc.minCol == -1 || cell.Col < sc.minCol {
			sc.minCol = cell.Col
		}
		if cell.Col > sc.maxCol {
			sc.maxCol = cell.Col
		}
	}
	sc.mu.Unlock()

	s.markDirty(tKey)
	return nil
}

// insertSorted inserts a value into a sorted slice maintaining order.
func insertSorted(slice []int, val int) []int {
	// Find insertion point
	i := 0
	for i < len(slice) && slice[i] < val {
		i++
	}
	// Already exists
	if i < len(slice) && slice[i] == val {
		return slice
	}
	// Insert
	slice = append(slice, 0)
	copy(slice[i+1:], slice[i:])
	slice[i] = val
	return slice
}

// BatchSet sets multiple cells.
func (s *Store) BatchSet(ctx context.Context, cellList []*cells.Cell) error {
	if len(cellList) == 0 {
		return nil
	}

	// Group cells by sheet, then by tile
	type sheetTileUpdates struct {
		tiles map[tileKey][]*cells.Cell
	}

	updates := make(map[string]*sheetTileUpdates)

	for _, cell := range cellList {
		tileRow, tileCol, _, _ := cellToTile(cell.Row, cell.Col)
		tKey := tileKey{cell.SheetID, tileRow, tileCol}

		stu := updates[cell.SheetID]
		if stu == nil {
			stu = &sheetTileUpdates{
				tiles: make(map[tileKey][]*cells.Cell),
			}
			updates[cell.SheetID] = stu
		}
		stu.tiles[tKey] = append(stu.tiles[tKey], cell)
	}

	// Collect dirty keys
	dirtyKeys := make([]tileKey, 0, len(updates)*2)

	// Apply updates per sheet
	for sheetID, stu := range updates {
		sc := s.getOrCreateSheetCache(sheetID)

		sc.mu.Lock()
		for tKey, tileCells := range stu.tiles {
			tile := sc.tiles[tKey]
			if tile == nil {
				tile = &Tile{Cells: make(map[string]*TileCell)}
				sc.tiles[tKey] = tile
			}

			for _, cell := range tileCells {
				_, _, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
				cellKey := tileCellKey(offsetRow, offsetCol)
				tc := cellToTileCell(cell)
				tile.Cells[cellKey] = tc

				// Update optimized indexes if sheet is fully loaded
				if sc.fullyLoaded {
					pos := cellPos{cell.Row, cell.Col}
					oldCell := sc.cellIndex[pos]
					sc.cellIndex[pos] = tc

					// Update rowIndex if this is a new cell
					if oldCell == nil {
						sc.rowIndex[cell.Row] = insertSorted(sc.rowIndex[cell.Row], cell.Col)
					}

					// Update bounds
					if sc.minRow == -1 || cell.Row < sc.minRow {
						sc.minRow = cell.Row
					}
					if cell.Row > sc.maxRow {
						sc.maxRow = cell.Row
					}
					if sc.minCol == -1 || cell.Col < sc.minCol {
						sc.minCol = cell.Col
					}
					if cell.Col > sc.maxCol {
						sc.maxCol = cell.Col
					}
				}
			}

			dirtyKeys = append(dirtyKeys, tKey)
		}
		sc.mu.Unlock()
	}

	// Mark all dirty
	s.mu.Lock()
	for _, key := range dirtyKeys {
		s.dirtyTiles[key] = struct{}{}
	}
	s.mu.Unlock()

	return nil
}

// Delete deletes a cell.
func (s *Store) Delete(ctx context.Context, sheetID string, row, col int) error {
	tileRow, tileCol, offsetRow, offsetCol := cellToTile(row, col)
	tKey := tileKey{sheetID, tileRow, tileCol}

	sc := s.getSheetCache(sheetID)
	if sc == nil {
		return s.underlying.Delete(ctx, sheetID, row, col)
	}

	sc.mu.Lock()
	tile := sc.tiles[tKey]
	if tile != nil {
		cellKey := tileCellKey(offsetRow, offsetCol)
		delete(tile.Cells, cellKey)

		// Update optimized indexes if sheet is fully loaded
		if sc.fullyLoaded {
			pos := cellPos{row, col}
			delete(sc.cellIndex, pos)
			sc.rowIndex[row] = removeFromSorted(sc.rowIndex[row], col)
			if len(sc.rowIndex[row]) == 0 {
				delete(sc.rowIndex, row)
			}
		}

		// If tile is now empty, remove it
		if len(tile.Cells) == 0 {
			delete(sc.tiles, tKey)
			s.markTileDeleted(tKey)
		} else {
			s.markDirty(tKey)
		}
	}
	sc.mu.Unlock()

	return nil
}

// removeFromSorted removes a value from a sorted slice.
func removeFromSorted(slice []int, val int) []int {
	for i := 0; i < len(slice); i++ {
		if slice[i] == val {
			return append(slice[:i], slice[i+1:]...)
		}
		if slice[i] > val {
			break
		}
	}
	return slice
}

// DeleteRange deletes cells in a range.
func (s *Store) DeleteRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	sc := s.getSheetCache(sheetID)
	if sc == nil {
		return s.underlying.DeleteRange(ctx, sheetID, startRow, startCol, endRow, endCol)
	}

	startTileRow := startRow / TileHeight
	endTileRow := endRow / TileHeight
	startTileCol := startCol / TileWidth
	endTileCol := endCol / TileWidth

	sc.mu.Lock()
	defer sc.mu.Unlock()

	for tr := startTileRow; tr <= endTileRow; tr++ {
		for tc := startTileCol; tc <= endTileCol; tc++ {
			tKey := tileKey{sheetID, tr, tc}
			tile := sc.tiles[tKey]
			if tile == nil {
				continue
			}

			// Delete cells in range
			for key := range tile.Cells {
				offsetRow, offsetCol := parseTileCellKey(key)
				cellRow := tr*TileHeight + offsetRow
				cellCol := tc*TileWidth + offsetCol

				if cellRow >= startRow && cellRow <= endRow &&
					cellCol >= startCol && cellCol <= endCol {
					delete(tile.Cells, key)

					// Update optimized indexes if sheet is fully loaded
					if sc.fullyLoaded {
						pos := cellPos{cellRow, cellCol}
						delete(sc.cellIndex, pos)
						sc.rowIndex[cellRow] = removeFromSorted(sc.rowIndex[cellRow], cellCol)
						if len(sc.rowIndex[cellRow]) == 0 {
							delete(sc.rowIndex, cellRow)
						}
					}
				}
			}

			if len(tile.Cells) == 0 {
				delete(sc.tiles, tKey)
				s.markTileDeleted(tKey)
			} else {
				s.markDirty(tKey)
			}
		}
	}

	return nil
}

// DeleteRowsRange deletes multiple rows and shifts remaining cells up.
func (s *Store) DeleteRowsRange(ctx context.Context, sheetID string, startRow, count int) error {
	// For shift operations, we flush and delegate to underlying
	if err := s.Flush(ctx); err != nil {
		return err
	}

	// Clear cache for this sheet (it will be repopulated on demand)
	s.sheets.Delete(sheetID)

	return s.underlying.DeleteRowsRange(ctx, sheetID, startRow, count)
}

// DeleteColsRange deletes multiple columns and shifts remaining cells left.
func (s *Store) DeleteColsRange(ctx context.Context, sheetID string, startCol, count int) error {
	if err := s.Flush(ctx); err != nil {
		return err
	}

	s.sheets.Delete(sheetID)

	return s.underlying.DeleteColsRange(ctx, sheetID, startCol, count)
}

// ShiftRows shifts rows (for insert/delete operations).
func (s *Store) ShiftRows(ctx context.Context, sheetID string, startRow, count int) error {
	if err := s.Flush(ctx); err != nil {
		return err
	}

	s.sheets.Delete(sheetID)

	return s.underlying.ShiftRows(ctx, sheetID, startRow, count)
}

// ShiftCols shifts columns (for insert/delete operations).
func (s *Store) ShiftCols(ctx context.Context, sheetID string, startCol, count int) error {
	if err := s.Flush(ctx); err != nil {
		return err
	}

	s.sheets.Delete(sheetID)

	return s.underlying.ShiftCols(ctx, sheetID, startCol, count)
}

// CreateMerge creates a merged region.
func (s *Store) CreateMerge(ctx context.Context, region *cells.MergedRegion) error {
	// Cache the merge
	v, _ := s.merges.LoadOrStore(region.SheetID, make([]*cells.MergedRegion, 0))
	regions := v.([]*cells.MergedRegion)
	regions = append(regions, region)
	s.merges.Store(region.SheetID, regions)

	s.markMergesDirty(region.SheetID)

	// Write through immediately for merges
	return s.underlying.CreateMerge(ctx, region)
}

// BatchCreateMerge creates multiple merged regions.
func (s *Store) BatchCreateMerge(ctx context.Context, regions []*cells.MergedRegion) error {
	if len(regions) == 0 {
		return nil
	}

	// Group by sheet
	bySheet := make(map[string][]*cells.MergedRegion)
	for _, r := range regions {
		bySheet[r.SheetID] = append(bySheet[r.SheetID], r)
	}

	// Cache
	for sheetID, sheetRegions := range bySheet {
		v, _ := s.merges.LoadOrStore(sheetID, make([]*cells.MergedRegion, 0))
		existing := v.([]*cells.MergedRegion)
		existing = append(existing, sheetRegions...)
		s.merges.Store(sheetID, existing)
	}

	// Write through immediately
	return s.underlying.BatchCreateMerge(ctx, regions)
}

// DeleteMerge deletes a merged region.
func (s *Store) DeleteMerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	// Remove from cache
	if v, ok := s.merges.Load(sheetID); ok {
		regions := v.([]*cells.MergedRegion)
		newRegions := make([]*cells.MergedRegion, 0, len(regions))
		for _, r := range regions {
			if !(r.StartRow == startRow && r.StartCol == startCol &&
				r.EndRow == endRow && r.EndCol == endCol) {
				newRegions = append(newRegions, r)
			}
		}
		s.merges.Store(sheetID, newRegions)
	}

	return s.underlying.DeleteMerge(ctx, sheetID, startRow, startCol, endRow, endCol)
}

// GetMergedRegions retrieves merged regions for a sheet.
func (s *Store) GetMergedRegions(ctx context.Context, sheetID string) ([]*cells.MergedRegion, error) {
	// Check cache first
	if v, ok := s.merges.Load(sheetID); ok {
		atomic.AddInt64(&s.stats.HitCount, 1)
		regions := v.([]*cells.MergedRegion)
		// Return a copy
		result := make([]*cells.MergedRegion, len(regions))
		copy(result, regions)
		return result, nil
	}

	// Load from underlying
	atomic.AddInt64(&s.stats.MissCount, 1)
	regions, err := s.underlying.GetMergedRegions(ctx, sheetID)
	if err != nil {
		return nil, err
	}

	// Cache
	s.merges.Store(sheetID, regions)

	return regions, nil
}

// Flush writes all dirty data to underlying store.
func (s *Store) Flush(ctx context.Context) error {
	// Get snapshot of dirty items
	s.mu.Lock()
	dirtyTiles := s.dirtyTiles
	deletedTiles := s.deletedTiles
	s.dirtyTiles = make(map[tileKey]struct{})
	s.deletedTiles = make(map[tileKey]struct{})
	s.mu.Unlock()

	if len(dirtyTiles) == 0 && len(deletedTiles) == 0 {
		return nil
	}

	atomic.AddInt64(&s.stats.FlushCount, 1)

	// Collect all cells from dirty tiles
	cellBatch := make([]*cells.Cell, 0, len(dirtyTiles)*100)

	for tKey := range dirtyTiles {
		sc := s.getSheetCache(tKey.sheetID)
		if sc == nil {
			continue
		}

		sc.mu.RLock()
		tile := sc.tiles[tKey]
		if tile != nil {
			for key, tc := range tile.Cells {
				if tc == nil {
					continue
				}
				offsetRow, offsetCol := parseTileCellKey(key)
				cellRow := tKey.tileRow*TileHeight + offsetRow
				cellCol := tKey.tileCol*TileWidth + offsetCol
				cellBatch = append(cellBatch, tileCellToCell(tc, tKey.sheetID, cellRow, cellCol))
			}
		}
		sc.mu.RUnlock()
	}

	// Write to underlying store
	if len(cellBatch) > 0 {
		if err := s.underlying.BatchSet(ctx, cellBatch); err != nil {
			// Re-mark as dirty on failure
			s.mu.Lock()
			for key := range dirtyTiles {
				s.dirtyTiles[key] = struct{}{}
			}
			s.mu.Unlock()
			return err
		}
	}

	// Handle deleted tiles by deleting ranges
	for tKey := range deletedTiles {
		startRow := tKey.tileRow * TileHeight
		startCol := tKey.tileCol * TileWidth
		endRow := startRow + TileHeight - 1
		endCol := startCol + TileWidth - 1
		_ = s.underlying.DeleteRange(ctx, tKey.sheetID, startRow, startCol, endRow, endCol)
	}

	return nil
}

// Preload loads specified sheets into cache from underlying store.
func (s *Store) Preload(ctx context.Context, sheetIDs []string) error {
	for _, sheetID := range sheetIDs {
		// Load cells (use a large range to get everything)
		cellList, err := s.underlying.GetRange(ctx, sheetID, 0, 0, 100000, 1000)
		if err != nil {
			return err
		}

		for _, cell := range cellList {
			_ = s.cacheCell(cell)
		}

		// Load merges
		regions, err := s.underlying.GetMergedRegions(ctx, sheetID)
		if err != nil {
			return err
		}
		s.merges.Store(sheetID, regions)
	}

	return nil
}

// GetStats returns cache statistics.
func (s *Store) GetStats() Stats {
	return Stats{
		HitCount:   atomic.LoadInt64(&s.stats.HitCount),
		MissCount:  atomic.LoadInt64(&s.stats.MissCount),
		FlushCount: atomic.LoadInt64(&s.stats.FlushCount),
	}
}

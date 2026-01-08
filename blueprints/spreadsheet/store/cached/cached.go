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

	// Optimized enables aggressive caching optimizations for best read performance.
	// When enabled:
	// - GetRange preloads entire tile ranges from underlying store on cache miss
	// - GetByPositions bulk-loads all required tiles on cache miss
	// - Uses optimized sorting algorithms for GetRange results
	// - Preloads sheets on first access
	Optimized bool
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

// sheetCache holds all tiles for a single sheet.
type sheetCache struct {
	mu           sync.RWMutex
	tiles        map[tileKey]*Tile
	loadedTiles  map[tileKey]bool // tracks which tiles have been loaded from underlying (for optimized mode)
	fullyLoaded  bool             // tracks if entire sheet has been loaded (for optimized mode)
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
	}
	actual, _ := s.sheets.LoadOrStore(sheetID, sc)
	return actual.(*sheetCache)
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
func (s *Store) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	// Optimized mode: preload the entire tile range if any tiles are missing
	if s.config.Optimized {
		return s.getRangeOptimized(ctx, sheetID, startRow, startCol, endRow, endCol)
	}

	sc := s.getSheetCache(sheetID)

	// If no cache, load from underlying
	if sc == nil {
		atomic.AddInt64(&s.stats.MissCount, 1)
		cellList, err := s.underlying.GetRange(ctx, sheetID, startRow, startCol, endRow, endCol)
		if err != nil {
			return nil, err
		}
		// Cache results
		for _, cell := range cellList {
			_ = s.cacheCell(cell)
		}
		return cellList, nil
	}

	atomic.AddInt64(&s.stats.HitCount, 1)

	// Calculate tile range
	startTileRow := startRow / TileHeight
	endTileRow := endRow / TileHeight
	startTileCol := startCol / TileWidth
	endTileCol := endCol / TileWidth

	// Pre-allocate result (estimate 10% density)
	capacity := (endRow - startRow + 1) * (endCol - startCol + 1) / 10
	if capacity < 16 {
		capacity = 16
	}
	result := make([]*cells.Cell, 0, capacity)

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Iterate through tiles in range
	for tr := startTileRow; tr <= endTileRow; tr++ {
		for tc := startTileCol; tc <= endTileCol; tc++ {
			tile := sc.tiles[tileKey{sheetID, tr, tc}]
			if tile == nil {
				continue
			}

			// Extract cells within the requested range
			for key, cell := range tile.Cells {
				if cell == nil {
					continue
				}
				offsetRow, offsetCol := parseTileCellKey(key)
				cellRow := tr*TileHeight + offsetRow
				cellCol := tc*TileWidth + offsetCol

				if cellRow >= startRow && cellRow <= endRow &&
					cellCol >= startCol && cellCol <= endCol {
					result = append(result, tileCellToCell(cell, sheetID, cellRow, cellCol))
				}
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

// getRangeOptimized is the optimized implementation of GetRange.
// It preloads all tiles in the range before reading to ensure cache hits.
func (s *Store) getRangeOptimized(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	sc := s.getOrCreateSheetCache(sheetID)

	// Calculate tile range
	startTileRow := startRow / TileHeight
	endTileRow := endRow / TileHeight
	startTileCol := startCol / TileWidth
	endTileCol := endCol / TileWidth

	// Check if we need to preload any tiles
	sc.mu.RLock()
	needsPreload := false
	for tr := startTileRow; tr <= endTileRow && !needsPreload; tr++ {
		for tc := startTileCol; tc <= endTileCol && !needsPreload; tc++ {
			tKey := tileKey{sheetID, tr, tc}
			if !sc.loadedTiles[tKey] {
				needsPreload = true
			}
		}
	}
	sc.mu.RUnlock()

	// Preload all tiles in the range at once from underlying store
	if needsPreload {
		atomic.AddInt64(&s.stats.MissCount, 1)

		// Calculate the full cell range covering all tiles
		tileStartRow := startTileRow * TileHeight
		tileEndRow := (endTileRow+1)*TileHeight - 1
		tileStartCol := startTileCol * TileWidth
		tileEndCol := (endTileCol+1)*TileWidth - 1

		cellList, err := s.underlying.GetRange(ctx, sheetID, tileStartRow, tileStartCol, tileEndRow, tileEndCol)
		if err != nil {
			return nil, err
		}

		// Bulk cache all cells
		sc.mu.Lock()
		for _, cell := range cellList {
			tileRow, tileCol, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
			tKey := tileKey{cell.SheetID, tileRow, tileCol}

			tile := sc.tiles[tKey]
			if tile == nil {
				tile = &Tile{Cells: make(map[string]*TileCell)}
				sc.tiles[tKey] = tile
			}
			cellKey := tileCellKey(offsetRow, offsetCol)
			tile.Cells[cellKey] = cellToTileCell(cell)
		}

		// Mark all tiles in range as loaded
		for tr := startTileRow; tr <= endTileRow; tr++ {
			for tc := startTileCol; tc <= endTileCol; tc++ {
				tKey := tileKey{sheetID, tr, tc}
				sc.loadedTiles[tKey] = true
				// Ensure tile exists even if empty
				if sc.tiles[tKey] == nil {
					sc.tiles[tKey] = &Tile{Cells: make(map[string]*TileCell)}
				}
			}
		}
		sc.mu.Unlock()
	} else {
		atomic.AddInt64(&s.stats.HitCount, 1)
	}

	// Now read from cache - all tiles are guaranteed to be loaded
	capacity := (endRow - startRow + 1) * (endCol - startCol + 1) / 10
	if capacity < 16 {
		capacity = 16
	}
	result := make([]*cells.Cell, 0, capacity)

	sc.mu.RLock()
	for tr := startTileRow; tr <= endTileRow; tr++ {
		for tc := startTileCol; tc <= endTileCol; tc++ {
			tile := sc.tiles[tileKey{sheetID, tr, tc}]
			if tile == nil {
				continue
			}

			for key, cell := range tile.Cells {
				if cell == nil {
					continue
				}
				offsetRow, offsetCol := parseTileCellKey(key)
				cellRow := tr*TileHeight + offsetRow
				cellCol := tc*TileWidth + offsetCol

				if cellRow >= startRow && cellRow <= endRow &&
					cellCol >= startCol && cellCol <= endCol {
					result = append(result, tileCellToCell(cell, sheetID, cellRow, cellCol))
				}
			}
		}
	}
	sc.mu.RUnlock()

	// Use efficient sort
	sortCells(result)

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
func (s *Store) GetByPositions(ctx context.Context, sheetID string, positions []cells.CellPosition) (map[cells.CellPosition]*cells.Cell, error) {
	if len(positions) == 0 {
		return make(map[cells.CellPosition]*cells.Cell), nil
	}

	// Optimized mode: preload all required tiles before reading
	if s.config.Optimized {
		return s.getByPositionsOptimized(ctx, sheetID, positions)
	}

	sc := s.getSheetCache(sheetID)

	// If no cache, load from underlying
	if sc == nil {
		atomic.AddInt64(&s.stats.MissCount, 1)
		result, err := s.underlying.GetByPositions(ctx, sheetID, positions)
		if err != nil {
			return nil, err
		}
		// Cache results
		for _, cell := range result {
			_ = s.cacheCell(cell)
		}
		return result, nil
	}

	atomic.AddInt64(&s.stats.HitCount, 1)

	result := make(map[cells.CellPosition]*cells.Cell, len(positions))

	// Group positions by tile
	positionsByTile := make(map[tileKey][]cells.CellPosition)
	for _, pos := range positions {
		tr, tc, _, _ := cellToTile(pos.Row, pos.Col)
		key := tileKey{sheetID, tr, tc}
		positionsByTile[key] = append(positionsByTile[key], pos)
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for tKey, positionsInTile := range positionsByTile {
		tile := sc.tiles[tKey]
		if tile == nil {
			continue
		}

		for _, pos := range positionsInTile {
			_, _, offsetRow, offsetCol := cellToTile(pos.Row, pos.Col)
			cellKey := tileCellKey(offsetRow, offsetCol)
			if tc := tile.Cells[cellKey]; tc != nil {
				result[pos] = tileCellToCell(tc, sheetID, pos.Row, pos.Col)
			}
		}
	}

	return result, nil
}

// getByPositionsOptimized is the optimized implementation of GetByPositions.
// It preloads all required tiles before reading to ensure cache hits.
func (s *Store) getByPositionsOptimized(ctx context.Context, sheetID string, positions []cells.CellPosition) (map[cells.CellPosition]*cells.Cell, error) {
	sc := s.getOrCreateSheetCache(sheetID)

	// Group positions by tile
	positionsByTile := make(map[tileKey][]cells.CellPosition)
	for _, pos := range positions {
		tr, tc, _, _ := cellToTile(pos.Row, pos.Col)
		key := tileKey{sheetID, tr, tc}
		positionsByTile[key] = append(positionsByTile[key], pos)
	}

	// Check which tiles need to be loaded
	sc.mu.RLock()
	tilesToLoad := make([]tileKey, 0)
	for tKey := range positionsByTile {
		if !sc.loadedTiles[tKey] {
			tilesToLoad = append(tilesToLoad, tKey)
		}
	}
	sc.mu.RUnlock()

	// Preload missing tiles
	if len(tilesToLoad) > 0 {
		atomic.AddInt64(&s.stats.MissCount, 1)

		// For each tile that needs loading, fetch its full range from underlying
		for _, tKey := range tilesToLoad {
			startRow := tKey.tileRow * TileHeight
			endRow := startRow + TileHeight - 1
			startCol := tKey.tileCol * TileWidth
			endCol := startCol + TileWidth - 1

			cellList, err := s.underlying.GetRange(ctx, sheetID, startRow, startCol, endRow, endCol)
			if err != nil {
				return nil, err
			}

			// Cache all cells from this tile
			sc.mu.Lock()
			tile := sc.tiles[tKey]
			if tile == nil {
				tile = &Tile{Cells: make(map[string]*TileCell)}
				sc.tiles[tKey] = tile
			}
			for _, cell := range cellList {
				_, _, offsetRow, offsetCol := cellToTile(cell.Row, cell.Col)
				cellKey := tileCellKey(offsetRow, offsetCol)
				tile.Cells[cellKey] = cellToTileCell(cell)
			}
			sc.loadedTiles[tKey] = true
			sc.mu.Unlock()
		}
	} else {
		atomic.AddInt64(&s.stats.HitCount, 1)
	}

	// Now read from cache - all tiles are guaranteed to be loaded
	result := make(map[cells.CellPosition]*cells.Cell, len(positions))

	sc.mu.RLock()
	for tKey, positionsInTile := range positionsByTile {
		tile := sc.tiles[tKey]
		if tile == nil {
			continue
		}

		for _, pos := range positionsInTile {
			_, _, offsetRow, offsetCol := cellToTile(pos.Row, pos.Col)
			cellKey := tileCellKey(offsetRow, offsetCol)
			if tc := tile.Cells[cellKey]; tc != nil {
				result[pos] = tileCellToCell(tc, sheetID, pos.Row, pos.Col)
			}
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
	tile.Cells[cellKey] = cellToTileCell(cell)
	sc.mu.Unlock()

	s.markDirty(tKey)
	return nil
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
				tile.Cells[cellKey] = cellToTileCell(cell)
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

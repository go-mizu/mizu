package main

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sort"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/oklog/ulid/v2"
)

// BenchConfig holds benchmark configuration.
type BenchConfig struct {
	Drivers    []string
	Categories []string
	Usecases   []string
	RunLoad    bool
	Quick      bool
	Verbose    bool
	Warmup     int
	Iterations int
}

// BenchResult represents a single benchmark result.
type BenchResult struct {
	Category    string
	Name        string
	Driver      string
	Operations  int64
	Duration    time.Duration
	NsPerOp     float64
	Throughput  float64 // ops/sec or cells/sec
	BytesPerOp  int64
	AllocsPerOp int64
	CellsPerOp  int
	Error       string

	// For load tests
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
	Max time.Duration
}

// BenchResults holds all benchmark results with metadata.
type BenchResults struct {
	Results       []BenchResult
	TotalDuration time.Duration
	SystemInfo    SystemInfo
	StartTime     time.Time
	EndTime       time.Time
}

// SystemInfo contains system information for the report.
type SystemInfo struct {
	OS         string
	Arch       string
	CPUs       int
	GoVersion  string
	GoMaxProcs int
}

// DriverContext holds initialized driver resources.
type DriverContext struct {
	Name       string
	DB         *sql.DB
	CellsStore cells.Store
	User       *users.User
	Workbook   *workbooks.Workbook
	Sheet      *sheets.Sheet
}

// BenchmarkRunner executes benchmarks across drivers.
type BenchmarkRunner struct {
	config   *BenchConfig
	drivers  map[string]*DriverContext
	results  []BenchResult
	registry *DriverRegistry
}

// NewBenchmarkRunner creates a new benchmark runner.
func NewBenchmarkRunner(config *BenchConfig) *BenchmarkRunner {
	return &BenchmarkRunner{
		config:   config,
		drivers:  make(map[string]*DriverContext),
		results:  []BenchResult{},
		registry: NewDriverRegistry(),
	}
}

// Run executes all configured benchmarks.
func (r *BenchmarkRunner) Run() (*BenchResults, error) {
	startTime := time.Now()

	// Get system info
	sysInfo := SystemInfo{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPUs:       runtime.NumCPU(),
		GoVersion:  runtime.Version(),
		GoMaxProcs: runtime.GOMAXPROCS(0),
	}

	// Initialize drivers
	drivers := r.getDriverList()
	for _, driverName := range drivers {
		if r.config.Verbose {
			fmt.Printf("Initializing driver: %s\n", driverName)
		}
		ctx, err := r.registry.SetupDriver(driverName)
		if err != nil {
			if r.config.Verbose {
				fmt.Printf("  Skipping %s: %v\n", driverName, err)
			}
			continue
		}
		r.drivers[driverName] = ctx
	}

	if len(r.drivers) == 0 {
		return nil, fmt.Errorf("no drivers available for testing")
	}

	defer r.cleanupDrivers()

	// Run benchmarks by category
	categories := r.getCategoryList()
	for _, cat := range categories {
		if r.config.Verbose {
			fmt.Printf("\n=== Category: %s ===\n", cat)
		}
		r.runCategory(cat)
	}

	// Run use case benchmarks
	usecases := r.getUsecaseList()
	for _, uc := range usecases {
		if r.config.Verbose {
			fmt.Printf("\n=== Use Case: %s ===\n", uc)
		}
		r.runUsecase(uc)
	}

	// Run load tests if requested
	if r.config.RunLoad {
		if r.config.Verbose {
			fmt.Printf("\n=== Load Tests ===\n")
		}
		r.runLoadTests()
	}

	endTime := time.Now()

	return &BenchResults{
		Results:       r.results,
		TotalDuration: endTime.Sub(startTime),
		SystemInfo:    sysInfo,
		StartTime:     startTime,
		EndTime:       endTime,
	}, nil
}

func (r *BenchmarkRunner) getDriverList() []string {
	if contains(r.config.Drivers, "all") {
		return []string{"duckdb", "sqlite", "swandb", "postgres"}
	}
	return r.config.Drivers
}

func (r *BenchmarkRunner) getCategoryList() []string {
	if contains(r.config.Categories, "all") {
		return []string{"cells", "rows", "merge", "format", "query"}
	}
	return r.config.Categories
}

func (r *BenchmarkRunner) getUsecaseList() []string {
	// If specific categories are requested (not "all"), don't run usecases by default
	if !contains(r.config.Categories, "all") && contains(r.config.Usecases, "all") {
		return []string{}
	}
	if contains(r.config.Usecases, "all") {
		if r.config.Quick {
			return []string{"financial", "import"} // Reduced for quick mode
		}
		return []string{"financial", "import", "report", "sparse", "bulk"}
	}
	return r.config.Usecases
}

func (r *BenchmarkRunner) runCategory(category string) {
	switch category {
	case "cells":
		r.runCellBenchmarks()
	case "rows":
		r.runRowBenchmarks()
	case "merge":
		r.runMergeBenchmarks()
	case "format":
		r.runFormatBenchmarks()
	case "query":
		r.runQueryBenchmarks()
	}
}

func (r *BenchmarkRunner) runUsecase(usecase string) {
	switch usecase {
	case "financial":
		r.runFinancialUsecase()
	case "import":
		r.runImportUsecase()
	case "report":
		r.runReportUsecase()
	case "sparse":
		r.runSparseUsecase()
	case "bulk":
		r.runBulkUsecase()
	}
}

// =============================================================================
// Cell Benchmarks
// =============================================================================

func (r *BenchmarkRunner) runCellBenchmarks() {
	sizes := []int{100, 500, 1000, 5000, 10000}
	if r.config.Quick {
		sizes = []int{100, 500}
	}

	// BatchSet benchmarks
	for _, size := range sizes {
		for name, ctx := range r.drivers {
			if r.config.Verbose {
				fmt.Printf("  Running BatchSet_%d for %s...\n", size, name)
			}
			result := r.benchmarkBatchSet(ctx, size)
			result.Category = "cells"
			result.Name = fmt.Sprintf("BatchSet_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}

	// Get benchmarks
	readSizes := []int{10, 50, 100}
	if r.config.Quick {
		readSizes = []int{10, 100}
	}

	for _, size := range readSizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkGetByPositionsSparse(ctx, size)
			result.Category = "cells"
			result.Name = fmt.Sprintf("GetByPositions_Sparse_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}

	// Dense reads
	denseSizes := []struct{ rows, cols int }{{10, 10}, {20, 20}, {50, 20}}
	if r.config.Quick {
		denseSizes = []struct{ rows, cols int }{{10, 10}, {20, 20}}
	}

	for _, size := range denseSizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkGetByPositionsDense(ctx, size.rows, size.cols)
			result.Category = "cells"
			result.Name = fmt.Sprintf("GetByPositions_Dense_%dx%d", size.rows, size.cols)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}

	// GetRange benchmarks
	rangeSizes := []struct{ rows, cols int }{{10, 10}, {100, 50}, {500, 100}}
	if r.config.Quick {
		rangeSizes = []struct{ rows, cols int }{{10, 10}, {100, 50}}
	}

	for _, size := range rangeSizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkGetRange(ctx, size.rows, size.cols)
			result.Category = "cells"
			result.Name = fmt.Sprintf("GetRange_%dx%d", size.rows, size.cols)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}
}

func (r *BenchmarkRunner) benchmarkBatchSet(ctx *DriverContext, count int) BenchResult {
	// Warmup
	for i := 0; i < r.config.Warmup; i++ {
		sheet := r.createSheet(ctx, i)
		cellList := generateCells(sheet.ID, count)
		_ = ctx.CellsStore.BatchSet(context.Background(), cellList)
	}

	// Benchmark
	var totalDuration time.Duration
	var totalAllocs int64
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, r.config.Warmup+i)
		cellList := generateCells(sheet.ID, count)

		runtime.GC()
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		err := ctx.CellsStore.BatchSet(context.Background(), cellList)
		elapsed := time.Since(start)

		runtime.ReadMemStats(&m2)
		totalAllocs += int64(m2.Mallocs - m1.Mallocs)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	nsPerOp := float64(avgDuration.Nanoseconds())
	throughput := float64(count) / avgDuration.Seconds()

	return BenchResult{
		Operations:  int64(iterations),
		Duration:    avgDuration,
		NsPerOp:     nsPerOp,
		Throughput:  throughput,
		CellsPerOp:  count,
		AllocsPerOp: totalAllocs / int64(iterations),
	}
}

func (r *BenchmarkRunner) benchmarkGetByPositionsSparse(ctx *DriverContext, count int) BenchResult {
	// Setup: create sparse cells
	sheet := r.createSheet(ctx, 1000)
	positions := make([]cells.CellPosition, count)
	for i := 0; i < count; i++ {
		row := i * 100
		col := i * 100
		cell := &cells.Cell{
			ID:        ulid.Make().String(),
			SheetID:   sheet.ID,
			Row:       row,
			Col:       col,
			Value:     fmt.Sprintf("sparse-%d", i),
			Display:   fmt.Sprintf("sparse-%d", i),
			Type:      cells.CellTypeText,
			UpdatedAt: time.Now(),
		}
		ctx.CellsStore.Set(context.Background(), cell)
		positions[i] = cells.CellPosition{Row: row, Col: col}
	}

	// Warmup
	for i := 0; i < r.config.Warmup; i++ {
		_, _ = ctx.CellsStore.GetByPositions(context.Background(), sheet.ID, positions)
	}

	// Benchmark
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := ctx.CellsStore.GetByPositions(context.Background(), sheet.ID, positions)
		elapsed := time.Since(start)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) benchmarkGetByPositionsDense(ctx *DriverContext, rows, cols int) BenchResult {
	// Setup: create dense grid
	sheet := r.createSheet(ctx, 2000)
	count := rows * cols
	cellList := make([]*cells.Cell, 0, count)
	positions := make([]cells.CellPosition, 0, count)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cellList = append(cellList, &cells.Cell{
				ID:        ulid.Make().String(),
				SheetID:   sheet.ID,
				Row:       row,
				Col:       col,
				Value:     fmt.Sprintf("dense-%d-%d", row, col),
				Display:   fmt.Sprintf("dense-%d-%d", row, col),
				Type:      cells.CellTypeText,
				UpdatedAt: time.Now(),
			})
			positions = append(positions, cells.CellPosition{Row: row, Col: col})
		}
	}
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	// Warmup
	for i := 0; i < r.config.Warmup; i++ {
		_, _ = ctx.CellsStore.GetByPositions(context.Background(), sheet.ID, positions)
	}

	// Benchmark
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := ctx.CellsStore.GetByPositions(context.Background(), sheet.ID, positions)
		elapsed := time.Since(start)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) benchmarkGetRange(ctx *DriverContext, rows, cols int) BenchResult {
	// Setup: create cells
	sheet := r.createSheet(ctx, 3000)
	count := rows * cols
	cellList := generateCellsForSheet(sheet.ID, rows, cols)
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	// Warmup
	for i := 0; i < r.config.Warmup; i++ {
		_, _ = ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, rows-1, cols-1)
	}

	// Benchmark
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, rows-1, cols-1)
		elapsed := time.Since(start)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

// =============================================================================
// Row Benchmarks
// =============================================================================

func (r *BenchmarkRunner) runRowBenchmarks() {
	sizes := []int{1, 10, 100, 1000}
	if r.config.Quick {
		sizes = []int{1, 100}
	}

	for _, size := range sizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkShiftRows(ctx, size)
			result.Category = "rows"
			result.Name = fmt.Sprintf("ShiftRows_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}

	colSizes := []int{1, 10, 50}
	if r.config.Quick {
		colSizes = []int{1, 10}
	}

	for _, size := range colSizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkShiftCols(ctx, size)
			result.Category = "rows"
			result.Name = fmt.Sprintf("ShiftCols_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}
}

func (r *BenchmarkRunner) benchmarkShiftRows(ctx *DriverContext, count int) BenchResult {
	// Setup: create sheet with data
	sheet := r.createSheet(ctx, 4000)
	cellList := generateCellsForSheet(sheet.ID, 100, 20) // 2000 cells
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	// Benchmark
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()
		err := ctx.CellsStore.ShiftRows(context.Background(), sheet.ID, 50, count)
		elapsed := time.Since(start)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) benchmarkShiftCols(ctx *DriverContext, count int) BenchResult {
	// Setup
	sheet := r.createSheet(ctx, 5000)
	cellList := generateCellsForSheet(sheet.ID, 100, 50) // 5000 cells
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	// Benchmark
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()
		err := ctx.CellsStore.ShiftCols(context.Background(), sheet.ID, 25, count)
		elapsed := time.Since(start)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: count,
	}
}

// =============================================================================
// Merge Benchmarks
// =============================================================================

func (r *BenchmarkRunner) runMergeBenchmarks() {
	sizes := []int{10, 50, 100}
	if r.config.Quick {
		sizes = []int{10, 50}
	}

	// Individual merges
	for _, size := range sizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkCreateMergeIndividual(ctx, size)
			result.Category = "merge"
			result.Name = fmt.Sprintf("CreateMerge_Individual_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}

	// Batch merges
	for _, size := range sizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkBatchCreateMerge(ctx, size)
			result.Category = "merge"
			result.Name = fmt.Sprintf("BatchCreateMerge_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}
}

func (r *BenchmarkRunner) benchmarkCreateMergeIndividual(ctx *DriverContext, count int) BenchResult {
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 6000+i)

		start := time.Now()
		for j := 0; j < count; j++ {
			region := &cells.MergedRegion{
				ID:       ulid.Make().String(),
				SheetID:  sheet.ID,
				StartRow: j * 3,
				StartCol: 0,
				EndRow:   j*3 + 1,
				EndCol:   1,
			}
			if err := ctx.CellsStore.CreateMerge(context.Background(), region); err != nil {
				return BenchResult{Error: err.Error()}
			}
		}
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) benchmarkBatchCreateMerge(ctx *DriverContext, count int) BenchResult {
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 7000+i)

		regions := make([]*cells.MergedRegion, count)
		for j := 0; j < count; j++ {
			regions[j] = &cells.MergedRegion{
				ID:       ulid.Make().String(),
				SheetID:  sheet.ID,
				StartRow: j * 3,
				StartCol: 0,
				EndRow:   j*3 + 1,
				EndCol:   1,
			}
		}

		start := time.Now()
		if err := ctx.CellsStore.BatchCreateMerge(context.Background(), regions); err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: count,
	}
}

// =============================================================================
// Format Benchmarks
// =============================================================================

func (r *BenchmarkRunner) runFormatBenchmarks() {
	count := 1000
	if r.config.Quick {
		count = 500
	}

	for name, ctx := range r.drivers {
		result := r.benchmarkBatchSetWithFormat(ctx, count)
		result.Category = "format"
		result.Name = "BatchSet_WithFormat"
		result.Driver = name
		r.results = append(r.results, result)
	}

	for name, ctx := range r.drivers {
		result := r.benchmarkBatchSetNoFormat(ctx, count)
		result.Category = "format"
		result.Name = "BatchSet_NoFormat"
		result.Driver = name
		r.results = append(r.results, result)
	}

	for name, ctx := range r.drivers {
		result := r.benchmarkBatchSetPartialFormat(ctx, count)
		result.Category = "format"
		result.Name = "BatchSet_PartialFormat"
		result.Driver = name
		r.results = append(r.results, result)
	}
}

func (r *BenchmarkRunner) benchmarkBatchSetWithFormat(ctx *DriverContext, count int) BenchResult {
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 8000+i)
		cellList := make([]*cells.Cell, count)

		for j := 0; j < count; j++ {
			cellList[j] = &cells.Cell{
				ID:      ulid.Make().String(),
				SheetID: sheet.ID,
				Row:     j / 100,
				Col:     j % 100,
				Value:   fmt.Sprintf("value-%d", j),
				Display: fmt.Sprintf("value-%d", j),
				Type:    cells.CellTypeText,
				Format: cells.Format{
					FontFamily:      "Arial",
					FontSize:        12,
					FontColor:       "#000000",
					Bold:            true,
					BackgroundColor: "#FFFFFF",
					HAlign:          "left",
					NumberFormat:    "#,##0.00",
				},
				Hyperlink: &cells.Hyperlink{
					URL:   "https://example.com",
					Label: "Example",
				},
				Note:      "Test note",
				UpdatedAt: time.Now(),
			}
		}

		start := time.Now()
		if err := ctx.CellsStore.BatchSet(context.Background(), cellList); err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) benchmarkBatchSetNoFormat(ctx *DriverContext, count int) BenchResult {
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 9000+i)
		cellList := generateCells(sheet.ID, count)

		start := time.Now()
		if err := ctx.CellsStore.BatchSet(context.Background(), cellList); err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) benchmarkBatchSetPartialFormat(ctx *DriverContext, count int) BenchResult {
	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 10000+i)
		cellList := make([]*cells.Cell, count)

		for j := 0; j < count; j++ {
			cellList[j] = &cells.Cell{
				ID:      ulid.Make().String(),
				SheetID: sheet.ID,
				Row:     j / 100,
				Col:     j % 100,
				Value:   fmt.Sprintf("value-%d", j),
				Display: fmt.Sprintf("value-%d", j),
				Type:    cells.CellTypeNumber,
				Format: cells.Format{
					NumberFormat: "#,##0.00",
				},
				UpdatedAt: time.Now(),
			}
		}

		start := time.Now()
		if err := ctx.CellsStore.BatchSet(context.Background(), cellList); err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

// =============================================================================
// Query Benchmarks
// =============================================================================

func (r *BenchmarkRunner) runQueryBenchmarks() {
	sizes := []int{1000, 5000, 10000}
	if r.config.Quick {
		sizes = []int{1000, 5000}
	}

	for _, size := range sizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkQueryNonEmpty(ctx, size)
			result.Category = "query"
			result.Name = fmt.Sprintf("Query_NonEmpty_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}
}

func (r *BenchmarkRunner) benchmarkQueryNonEmpty(ctx *DriverContext, count int) BenchResult {
	// Setup: create sheet with sparse data
	sheet := r.createSheet(ctx, 11000)
	cellList := generateCells(sheet.ID, count)
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()
		// Query a range that covers all cells
		_, err := ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, count/100+1, 100)
		elapsed := time.Since(start)

		if err != nil {
			return BenchResult{Error: err.Error()}
		}
		totalDuration += elapsed
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: count,
	}
}

// =============================================================================
// Use Case Benchmarks
// =============================================================================

func (r *BenchmarkRunner) runFinancialUsecase() {
	// Financial modeling: 500 rows × 50 cols, 30% formulas, heavy formatting
	for name, ctx := range r.drivers {
		result := r.benchmarkFinancialWorkbook(ctx)
		result.Category = "usecase"
		result.Name = "Financial_Workbook"
		result.Driver = name
		r.results = append(r.results, result)
	}
}

func (r *BenchmarkRunner) benchmarkFinancialWorkbook(ctx *DriverContext) BenchResult {
	rows, cols := 500, 50
	if r.config.Quick {
		rows, cols = 100, 20
	}
	count := rows * cols

	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 12000+i)
		cellList := make([]*cells.Cell, 0, count)

		for row := 0; row < rows; row++ {
			for col := 0; col < cols; col++ {
				cell := &cells.Cell{
					ID:        ulid.Make().String(),
					SheetID:   sheet.ID,
					Row:       row,
					Col:       col,
					UpdatedAt: time.Now(),
				}

				// 30% formulas
				if (row*cols+col)%10 < 3 {
					cell.Formula = fmt.Sprintf("=SUM(A%d:A%d)", row, row+10)
					cell.Display = "1234.56"
					cell.Type = cells.CellTypeFormula
				} else {
					cell.Value = float64(row*100 + col)
					cell.Display = fmt.Sprintf("%.2f", float64(row*100+col))
					cell.Type = cells.CellTypeNumber
				}

				// Heavy formatting
				cell.Format = cells.Format{
					FontFamily:   "Arial",
					FontSize:     11,
					FontColor:    "#333333",
					NumberFormat: "$#,##0.00",
					HAlign:       "right",
				}
				if col == 0 {
					cell.Format.Bold = true
				}

				cellList = append(cellList, cell)
			}
		}

		start := time.Now()
		if err := ctx.CellsStore.BatchSet(context.Background(), cellList); err != nil {
			return BenchResult{Error: err.Error()}
		}

		// Simulate viewport read (typical 50 rows × 20 cols visible)
		_, _ = ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, 49, 19)

		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(count) / avgDuration.Seconds(),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) runImportUsecase() {
	sizes := []int{10000, 50000, 100000}
	if r.config.Quick {
		sizes = []int{10000, 50000}
	}

	for _, size := range sizes {
		for name, ctx := range r.drivers {
			result := r.benchmarkCSVImport(ctx, size)
			result.Category = "usecase"
			result.Name = fmt.Sprintf("Import_CSV_%d", size)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}
}

func (r *BenchmarkRunner) benchmarkCSVImport(ctx *DriverContext, totalCells int) BenchResult {
	cols := 20
	rows := totalCells / cols
	batchSize := 1000

	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 13000+i)

		start := time.Now()

		// Simulate CSV import with batching
		for batchStart := 0; batchStart < rows; batchStart += batchSize / cols {
			batchEnd := batchStart + batchSize/cols
			if batchEnd > rows {
				batchEnd = rows
			}

			batch := make([]*cells.Cell, 0, (batchEnd-batchStart)*cols)
			for row := batchStart; row < batchEnd; row++ {
				for col := 0; col < cols; col++ {
					batch = append(batch, &cells.Cell{
						ID:        ulid.Make().String(),
						SheetID:   sheet.ID,
						Row:       row,
						Col:       col,
						Value:     fmt.Sprintf("csv-%d-%d", row, col),
						Display:   fmt.Sprintf("csv-%d-%d", row, col),
						Type:      cells.CellTypeText,
						UpdatedAt: time.Now(),
					})
				}
			}

			if err := ctx.CellsStore.BatchSet(context.Background(), batch); err != nil {
				return BenchResult{Error: err.Error()}
			}
		}

		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(totalCells) / avgDuration.Seconds(),
		CellsPerOp: totalCells,
	}
}

func (r *BenchmarkRunner) runReportUsecase() {
	for name, ctx := range r.drivers {
		result := r.benchmarkReportGeneration(ctx)
		result.Category = "usecase"
		result.Name = "Report_Generation"
		result.Driver = name
		r.results = append(r.results, result)
	}
}

func (r *BenchmarkRunner) benchmarkReportGeneration(ctx *DriverContext) BenchResult {
	// Setup: large dataset
	rows, cols := 10000, 30
	if r.config.Quick {
		rows = 1000
	}
	count := rows * cols

	sheet := r.createSheet(ctx, 14000)
	cellList := generateCellsForSheet(sheet.ID, rows, cols)
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Simulate report generation: multiple range queries
		// Summary range (first 100 rows)
		_, _ = ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, 99, cols-1)

		// Detail range (random section)
		_, _ = ctx.CellsStore.GetRange(context.Background(), sheet.ID, 500, 0, 599, cols-1)

		// Full data export
		_, _ = ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, rows-1, cols-1)

		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: count,
	}
}

func (r *BenchmarkRunner) runSparseUsecase() {
	for name, ctx := range r.drivers {
		result := r.benchmarkSparseData(ctx)
		result.Category = "usecase"
		result.Name = "Sparse_Data"
		result.Driver = name
		r.results = append(r.results, result)
	}
}

func (r *BenchmarkRunner) benchmarkSparseData(ctx *DriverContext) BenchResult {
	// Sparse: 10K rows × 200 cols, 10% density
	rows, cols := 10000, 200
	density := 0.10
	if r.config.Quick {
		rows = 1000
	}
	totalCells := int(float64(rows*cols) * density)

	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 15000+i)

		// Generate sparse cells
		cellList := make([]*cells.Cell, 0, totalCells)
		step := int(1.0 / density)
		for idx := 0; idx < totalCells; idx++ {
			row := (idx * step) / cols
			col := (idx * step) % cols
			cellList = append(cellList, &cells.Cell{
				ID:        ulid.Make().String(),
				SheetID:   sheet.ID,
				Row:       row,
				Col:       col,
				Value:     fmt.Sprintf("sparse-%d", idx),
				Display:   fmt.Sprintf("sparse-%d", idx),
				Type:      cells.CellTypeText,
				UpdatedAt: time.Now(),
			})
		}

		start := time.Now()
		if err := ctx.CellsStore.BatchSet(context.Background(), cellList); err != nil {
			return BenchResult{Error: err.Error()}
		}

		// Random access reads
		positions := make([]cells.CellPosition, 100)
		for j := 0; j < 100; j++ {
			positions[j] = cells.CellPosition{Row: j * 100, Col: j % cols}
		}
		_, _ = ctx.CellsStore.GetByPositions(context.Background(), sheet.ID, positions)

		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		Throughput: float64(totalCells) / avgDuration.Seconds(),
		CellsPerOp: totalCells,
	}
}

func (r *BenchmarkRunner) runBulkUsecase() {
	for name, ctx := range r.drivers {
		result := r.benchmarkBulkOperations(ctx)
		result.Category = "usecase"
		result.Name = "Bulk_Operations"
		result.Driver = name
		r.results = append(r.results, result)
	}
}

func (r *BenchmarkRunner) benchmarkBulkOperations(ctx *DriverContext) BenchResult {
	// Bulk ops: insert rows, delete rows, merges
	rows, cols := 1000, 50
	if r.config.Quick {
		rows = 200
	}

	var totalDuration time.Duration
	iterations := r.config.Iterations

	for i := 0; i < iterations; i++ {
		sheet := r.createSheet(ctx, 16000+i)
		cellList := generateCellsForSheet(sheet.ID, rows, cols)
		ctx.CellsStore.BatchSet(context.Background(), cellList)

		start := time.Now()

		// Insert 10 rows in middle
		_ = ctx.CellsStore.ShiftRows(context.Background(), sheet.ID, rows/2, 10)

		// Delete 5 rows
		_ = ctx.CellsStore.ShiftRows(context.Background(), sheet.ID, rows/4, -5)

		// Create 20 merge regions
		regions := make([]*cells.MergedRegion, 20)
		for j := 0; j < 20; j++ {
			regions[j] = &cells.MergedRegion{
				ID:       ulid.Make().String(),
				SheetID:  sheet.ID,
				StartRow: j * 5,
				StartCol: 0,
				EndRow:   j*5 + 1,
				EndCol:   2,
			}
		}
		_ = ctx.CellsStore.BatchCreateMerge(context.Background(), regions)

		// Delete range
		_ = ctx.CellsStore.DeleteRange(context.Background(), sheet.ID, 0, cols-5, 10, cols-1)

		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	return BenchResult{
		Operations: int64(iterations),
		Duration:   avgDuration,
		NsPerOp:    float64(avgDuration.Nanoseconds()),
		CellsPerOp: rows * cols,
	}
}

// =============================================================================
// Load Tests
// =============================================================================

func (r *BenchmarkRunner) runLoadTests() {
	// L1: Sustained write load
	rates := []int{100, 500, 1000}
	if r.config.Quick {
		rates = []int{100, 500}
	}

	for _, rate := range rates {
		for name, ctx := range r.drivers {
			result := r.benchmarkSustainedWrite(ctx, rate)
			result.Category = "load"
			result.Name = fmt.Sprintf("Sustained_Write_%d_cps", rate)
			result.Driver = name
			r.results = append(r.results, result)
		}
	}

	// L2: Mixed workload
	for name, ctx := range r.drivers {
		result := r.benchmarkMixedWorkload(ctx)
		result.Category = "load"
		result.Name = "Mixed_Workload"
		result.Driver = name
		r.results = append(r.results, result)
	}
}

func (r *BenchmarkRunner) benchmarkSustainedWrite(ctx *DriverContext, cellsPerSecond int) BenchResult {
	duration := 10 * time.Second
	if r.config.Quick {
		duration = 3 * time.Second
	}

	sheet := r.createSheet(ctx, 17000)
	batchSize := 50
	batchInterval := time.Duration(float64(time.Second) * float64(batchSize) / float64(cellsPerSecond))

	var latencies []time.Duration
	var totalCells int
	var errors int

	start := time.Now()
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	row := 0
	for time.Since(start) < duration {
		<-ticker.C

		batch := make([]*cells.Cell, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = &cells.Cell{
				ID:        ulid.Make().String(),
				SheetID:   sheet.ID,
				Row:       row,
				Col:       j,
				Value:     fmt.Sprintf("load-%d-%d", row, j),
				Display:   fmt.Sprintf("load-%d-%d", row, j),
				Type:      cells.CellTypeText,
				UpdatedAt: time.Now(),
			}
		}

		opStart := time.Now()
		err := ctx.CellsStore.BatchSet(context.Background(), batch)
		latency := time.Since(opStart)

		if err != nil {
			errors++
		} else {
			latencies = append(latencies, latency)
			totalCells += batchSize
		}
		row++
	}

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var p50, p95, p99, maxLatency time.Duration
	if len(latencies) > 0 {
		p50 = latencies[len(latencies)*50/100]
		p95 = latencies[len(latencies)*95/100]
		p99 = latencies[len(latencies)*99/100]
		maxLatency = latencies[len(latencies)-1]
	}

	actualDuration := time.Since(start)
	return BenchResult{
		Operations: int64(len(latencies)),
		Duration:   actualDuration,
		Throughput: float64(totalCells) / actualDuration.Seconds(),
		CellsPerOp: totalCells,
		P50:        p50,
		P95:        p95,
		P99:        p99,
		Max:        maxLatency,
	}
}

func (r *BenchmarkRunner) benchmarkMixedWorkload(ctx *DriverContext) BenchResult {
	duration := 10 * time.Second
	if r.config.Quick {
		duration = 3 * time.Second
	}

	// Setup: create sheet with data
	sheet := r.createSheet(ctx, 18000)
	cellList := generateCellsForSheet(sheet.ID, 100, 50) // 5000 cells
	ctx.CellsStore.BatchSet(context.Background(), cellList)

	var latencies []time.Duration
	var totalOps int

	start := time.Now()

	for time.Since(start) < duration {
		opType := totalOps % 10 // 60% read, 30% write, 10% range

		opStart := time.Now()
		var err error

		switch {
		case opType < 6: // Read
			positions := []cells.CellPosition{{Row: totalOps % 100, Col: totalOps % 50}}
			_, err = ctx.CellsStore.GetByPositions(context.Background(), sheet.ID, positions)
		case opType < 9: // Write
			cell := &cells.Cell{
				ID:        ulid.Make().String(),
				SheetID:   sheet.ID,
				Row:       totalOps % 100,
				Col:       totalOps % 50,
				Value:     fmt.Sprintf("mixed-%d", totalOps),
				Display:   fmt.Sprintf("mixed-%d", totalOps),
				Type:      cells.CellTypeText,
				UpdatedAt: time.Now(),
			}
			err = ctx.CellsStore.Set(context.Background(), cell)
		default: // Range
			_, err = ctx.CellsStore.GetRange(context.Background(), sheet.ID, 0, 0, 10, 10)
		}

		latency := time.Since(opStart)
		if err == nil {
			latencies = append(latencies, latency)
		}
		totalOps++
	}

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var p50, p95, p99, maxLatency time.Duration
	if len(latencies) > 0 {
		p50 = latencies[len(latencies)*50/100]
		p95 = latencies[len(latencies)*95/100]
		p99 = latencies[len(latencies)*99/100]
		maxLatency = latencies[len(latencies)-1]
	}

	actualDuration := time.Since(start)
	return BenchResult{
		Operations: int64(totalOps),
		Duration:   actualDuration,
		Throughput: float64(totalOps) / actualDuration.Seconds(),
		P50:        p50,
		P95:        p95,
		P99:        p99,
		Max:        maxLatency,
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func (r *BenchmarkRunner) createSheet(ctx *DriverContext, index int) *sheets.Sheet {
	sheetID := ulid.Make().String()
	now := time.Now()

	_, _ = ctx.DB.ExecContext(context.Background(), `
		INSERT INTO sheets (id, workbook_id, name, index_num, hidden, grid_color,
			frozen_rows, frozen_cols, default_row_height, default_col_width,
			row_heights, col_widths, hidden_rows, hidden_cols, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, sheetID, ctx.Workbook.ID, fmt.Sprintf("BenchSheet%d", index), index, false, "#E2E8F0",
		0, 0, 21, 100, "{}", "{}", "[]", "[]", now, now)

	return &sheets.Sheet{ID: sheetID, WorkbookID: ctx.Workbook.ID}
}

func (r *BenchmarkRunner) cleanupDrivers() {
	for _, ctx := range r.drivers {
		if ctx.DB != nil {
			ctx.DB.Close()
		}
	}
}

func generateCells(sheetID string, count int) []*cells.Cell {
	cellList := make([]*cells.Cell, count)
	now := time.Now()
	cols := 100

	for i := 0; i < count; i++ {
		cellList[i] = &cells.Cell{
			ID:        ulid.Make().String(),
			SheetID:   sheetID,
			Row:       i / cols,
			Col:       i % cols,
			Value:     fmt.Sprintf("value-%d", i),
			Display:   fmt.Sprintf("value-%d", i),
			Type:      cells.CellTypeText,
			UpdatedAt: now,
		}
	}

	return cellList
}

func generateCellsForSheet(sheetID string, rows, cols int) []*cells.Cell {
	cellList := make([]*cells.Cell, 0, rows*cols)
	now := time.Now()

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cellList = append(cellList, &cells.Cell{
				ID:        ulid.Make().String(),
				SheetID:   sheetID,
				Row:       row,
				Col:       col,
				Value:     fmt.Sprintf("cell-%d-%d", row, col),
				Display:   fmt.Sprintf("cell-%d-%d", row, col),
				Type:      cells.CellTypeText,
				UpdatedAt: now,
			})
		}
	}

	return cellList
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

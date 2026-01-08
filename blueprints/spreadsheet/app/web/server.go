package web

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/app/web/handler"
	"github.com/go-mizu/blueprints/spreadsheet/app/web/handler/api"
	"github.com/go-mizu/blueprints/spreadsheet/assets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/export"
	"github.com/go-mizu/blueprints/spreadsheet/feature/importer"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/go-mizu/blueprints/spreadsheet/pkg/password"
	"github.com/go-mizu/blueprints/spreadsheet/store/duckdb"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app *mizu.App
	cfg Config
	db  *sql.DB

	// Services
	users     users.API
	workbooks workbooks.API
	sheets    sheets.API
	cells     cells.API
	charts    charts.API
	exporter  export.API
	importer  importer.API

	// Handlers
	authHandlers         *api.Auth
	workbookHandlers     *api.Workbook
	sheetHandlers        *api.Sheet
	cellHandlers         *api.Cell
	chartsHandlers       *api.Charts
	importExportHandlers *api.ImportExport
	uiHandlers           *handler.UI
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "spreadsheet.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create core store for schema initialization
	coreStore, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Initialize schema
	if err := coreStore.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create feature stores
	usersStore := duckdb.NewUsersStore(db)
	workbooksStore := duckdb.NewWorkbooksStore(db)
	sheetsStore := duckdb.NewSheetsStore(db)
	cellsStore := duckdb.NewCellsStore(db)
	chartsStore := duckdb.NewChartsStore(db)

	// Create services
	usersSvc := users.NewService(usersStore)
	workbooksSvc := workbooks.NewService(workbooksStore)
	sheetsSvc := sheets.NewService(sheetsStore)
	cellsSvc := cells.NewService(cellsStore, usersSvc.GetSecret())

	// Create cell data provider adapter for charts
	cellProvider := &cellDataProviderAdapter{cells: cellsSvc}
	chartsSvc := charts.NewService(chartsStore, cellProvider)

	// Wire up sheet resolver for cross-sheet formula evaluation
	cellsSvc.SetSheetResolver(sheetsSvc)

	// Create dev user in dev mode
	if cfg.Dev {
		ctx := context.Background()
		// Check if dev user exists
		if _, err := usersStore.GetByID(ctx, devUserID); err != nil {
			// Create dev user
			now := time.Now()
			hash, _ := password.Hash("password")
			devUser := &users.User{
				ID:        devUserID,
				Email:     "dev@example.com",
				Name:      "Developer",
				Password:  hash,
				CreatedAt: now,
				UpdatedAt: now,
			}
			usersStore.Create(ctx, devUser)
			slog.Info("Created dev user", "id", devUserID, "email", "dev@example.com")

			// Create a sample workbook for the dev user
			wb, err := workbooksSvc.Create(ctx, &workbooks.CreateIn{
				Name:      "Sample Spreadsheet",
				OwnerID:   devUserID,
				CreatedBy: devUserID,
			})
			if err == nil {
				// Create a sheet
				sheet, err := sheetsSvc.Create(ctx, &sheets.CreateIn{
					WorkbookID: wb.ID,
					Name:       "Sheet1",
					Index:      0,
					CreatedBy:  devUserID,
				})
				if err == nil {
					// Add sample data
					sampleData := [][]interface{}{
						{"Product", "Q1", "Q2", "Q3", "Q4", "Total"},
						{"Laptops", 15000, 18000, 22000, 25000, "=SUM(B2:E2)"},
						{"Phones", 25000, 28000, 32000, 35000, "=SUM(B3:E3)"},
						{"Tablets", 8000, 9500, 11000, 12500, "=SUM(B4:E4)"},
						{"Total", "=SUM(B2:B4)", "=SUM(C2:C4)", "=SUM(D2:D4)", "=SUM(E2:E4)", "=SUM(F2:F4)"},
					}
					for row, rowData := range sampleData {
						for col, value := range rowData {
							var formula string
							var cellValue interface{}
							if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
								formula = strVal
							} else {
								cellValue = value
							}
							cellsSvc.Set(ctx, sheet.ID, row, col, &cells.SetCellIn{
								Value:   cellValue,
								Formula: formula,
							})
						}
					}
					slog.Info("Created sample workbook", "id", wb.ID)
				}

				// Create Charts demo sheet
				chartsSheet, err := sheetsSvc.Create(ctx, &sheets.CreateIn{
					WorkbookID: wb.ID,
					Name:       "Charts",
					Index:      1,
					CreatedBy:  devUserID,
				})
				if err == nil {
					// Add demo data for charts
					chartsDemoData := [][]interface{}{
						// Monthly Sales Data (A1:E7) - Row 0-6
						{"Month", "Sales", "Expenses", "Profit", "Growth"},
						{"Jan", 12000, 8000, 4000, 0},
						{"Feb", 15000, 9000, 6000, 50},
						{"Mar", 18000, 10000, 8000, 33},
						{"Apr", 14000, 9500, 4500, -44},
						{"May", 20000, 11000, 9000, 100},
						{"Jun", 25000, 12000, 13000, 44},
						// Blank row - Row 7
						{},
						// Regional Data (A9:C13) - Row 8-12
						{"Region", "Q1 Sales", "Q2 Sales"},
						{"North", 45000, 52000},
						{"South", 38000, 41000},
						{"East", 32000, 45000},
						{"West", 28000, 35000},
						// Blank row - Row 13
						{},
						// Category Distribution (A15:B19) - Row 14-18
						{"Category", "Market Share"},
						{"Product A", 35},
						{"Product B", 25},
						{"Product C", 22},
						{"Product D", 18},
						// Blank row - Row 19
						{},
						// Radar Chart Data (A21:F26) - Row 20-25
						{"Metric", "Team A", "Team B", "Team C", "Team D", "Team E"},
						{"Speed", 85, 72, 90, 68, 78},
						{"Quality", 78, 88, 75, 92, 82},
						{"Efficiency", 92, 75, 80, 85, 70},
						{"Innovation", 70, 82, 88, 75, 90},
						{"Collaboration", 88, 90, 72, 80, 85},
						// Blank row - Row 26
						{},
						// Scatter Plot Data (A28:C38) - Row 27-37
						{"Item", "X Value", "Y Value"},
						{"Point 1", 10, 25},
						{"Point 2", 15, 35},
						{"Point 3", 22, 42},
						{"Point 4", 28, 38},
						{"Point 5", 35, 55},
						{"Point 6", 42, 48},
						{"Point 7", 50, 65},
						{"Point 8", 58, 72},
						{"Point 9", 65, 78},
						{"Point 10", 75, 85},
						// Blank row - Row 38
						{},
						// Stacked Data (A40:D45) - Row 39-44
						{"Quarter", "Hardware", "Software", "Services"},
						{"Q1", 25000, 35000, 18000},
						{"Q2", 28000, 40000, 22000},
						{"Q3", 32000, 45000, 25000},
						{"Q4", 38000, 52000, 30000},
					}
					for row, rowData := range chartsDemoData {
						for col, value := range rowData {
							if value == nil {
								continue
							}
							var formula string
							var cellValue interface{}
							if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
								formula = strVal
							} else {
								cellValue = value
							}
							cellsSvc.Set(ctx, chartsSheet.ID, row, col, &cells.SetCellIn{
								Value:   cellValue,
								Formula: formula,
							})
						}
					}

					// Create demo charts
					// 1. Line Chart - Monthly Sales Trend
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Sales Trend",
						ChartType: charts.ChartTypeLine,
						Position:  charts.Position{Row: 0, Col: 6, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 500, Height: 300},
						DataRanges: []charts.DataRange{{
							StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Monthly Sales Trend", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Axes: &charts.AxesConfig{
							XAxis: &charts.AxisConfig{GridLines: false},
							YAxis: &charts.AxisConfig{GridLines: true, Title: &charts.ChartTitle{Text: "Amount ($)"}},
						},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 2. Column Chart - Regional Comparison
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Regional Sales",
						ChartType: charts.ChartTypeColumn,
						Position:  charts.Position{Row: 0, Col: 12, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 450, Height: 300},
						DataRanges: []charts.DataRange{{
							StartRow: 8, StartCol: 0, EndRow: 12, EndCol: 2,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Regional Sales Comparison", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 3. Pie Chart - Market Share
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Market Share",
						ChartType: charts.ChartTypePie,
						Position:  charts.Position{Row: 16, Col: 6, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 400, Height: 350},
						DataRanges: []charts.DataRange{{
							StartRow: 14, StartCol: 0, EndRow: 18, EndCol: 1,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Product Market Share", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "right"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 4. Doughnut Chart - Same data as pie
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Market Share (Doughnut)",
						ChartType: charts.ChartTypeDoughnut,
						Position:  charts.Position{Row: 16, Col: 12, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 400, Height: 350},
						DataRanges: []charts.DataRange{{
							StartRow: 14, StartCol: 0, EndRow: 18, EndCol: 1,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Market Share (Doughnut)", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "right"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true, CutoutPercentage: 50},
					})

					// 5. Bar Chart - Regional Comparison (Horizontal)
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Regional Bar",
						ChartType: charts.ChartTypeBar,
						Position:  charts.Position{Row: 32, Col: 6, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 450, Height: 300},
						DataRanges: []charts.DataRange{{
							StartRow: 8, StartCol: 0, EndRow: 12, EndCol: 2,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Regional Sales (Horizontal)", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 6. Area Chart - Sales Trend
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Sales Area",
						ChartType: charts.ChartTypeArea,
						Position:  charts.Position{Row: 32, Col: 12, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 500, Height: 300},
						DataRanges: []charts.DataRange{{
							StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Sales, Expenses & Profit (Area)", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 7. Stacked Column Chart
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Revenue Breakdown",
						ChartType: charts.ChartTypeStackedColumn,
						Position:  charts.Position{Row: 48, Col: 6, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 500, Height: 320},
						DataRanges: []charts.DataRange{{
							StartRow: 39, StartCol: 0, EndRow: 43, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Quarterly Revenue by Segment", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 8. Radar Chart - Team Performance
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Team Performance",
						ChartType: charts.ChartTypeRadar,
						Position:  charts.Position{Row: 48, Col: 12, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 450, Height: 350},
						DataRanges: []charts.DataRange{{
							StartRow: 20, StartCol: 0, EndRow: 25, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Team Performance Comparison", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 9. Scatter Chart - Correlation
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "XY Correlation",
						ChartType: charts.ChartTypeScatter,
						Position:  charts.Position{Row: 64, Col: 6, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 480, Height: 320},
						DataRanges: []charts.DataRange{{
							StartRow: 27, StartCol: 1, EndRow: 37, EndCol: 2,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "X vs Y Correlation", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: false, Position: "none"},
						Axes: &charts.AxesConfig{
							XAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "X Value"}, GridLines: true},
							YAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Y Value"}, GridLines: true},
						},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 10. Stacked Bar Chart (Horizontal Stacked)
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Revenue Stacked Bar",
						ChartType: charts.ChartTypeStackedBar,
						Position:  charts.Position{Row: 64, Col: 12, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 500, Height: 320},
						DataRanges: []charts.DataRange{{
							StartRow: 39, StartCol: 0, EndRow: 43, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Revenue by Segment (Horizontal)", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 11. Stacked Area Chart
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Sales Stacked Area",
						ChartType: charts.ChartTypeStackedArea,
						Position:  charts.Position{Row: 80, Col: 6, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 500, Height: 300},
						DataRanges: []charts.DataRange{{
							StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Sales Trend (Stacked Area)", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					// 12. Combo Chart - Mixed Line and Column
					chartsSvc.Create(ctx, &charts.CreateIn{
						SheetID:   chartsSheet.ID,
						Name:      "Sales Combo",
						ChartType: charts.ChartTypeCombo,
						Position:  charts.Position{Row: 80, Col: 12, OffsetX: 0, OffsetY: 0},
						Size:      charts.Size{Width: 500, Height: 300},
						DataRanges: []charts.DataRange{{
							StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
							HasHeader: true,
						}},
						Title: &charts.ChartTitle{Text: "Sales vs Profit (Combo)", FontSize: 16, Bold: true},
						Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
						Series: []charts.SeriesConfig{
							{Name: "Sales", ChartType: charts.ChartTypeColumn, Color: "#4CAF50"},
							{Name: "Expenses", ChartType: charts.ChartTypeColumn, Color: "#FF9800"},
							{Name: "Profit", ChartType: charts.ChartTypeLine, Color: "#2196F3", BorderWidth: 3},
						},
						Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
					})

					slog.Info("Created Charts demo sheet with 12 chart types", "id", chartsSheet.ID)
				}
			}
		}
	}

	// Create export and import services
	exportSvc := export.NewService(workbooksSvc, sheetsSvc, cellsSvc)
	importSvc := importer.NewService(workbooksSvc, sheetsSvc, cellsSvc)

	s := &Server{
		app:       mizu.New(),
		cfg:       cfg,
		db:        db,
		users:     usersSvc,
		workbooks: workbooksSvc,
		sheets:    sheetsSvc,
		cells:     cellsSvc,
		charts:    chartsSvc,
		exporter:  exportSvc,
		importer:  importSvc,
	}

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	// Create handlers
	s.authHandlers = api.NewAuth(usersSvc)
	s.workbookHandlers = api.NewWorkbook(workbooksSvc, sheetsSvc, s.getUserID)
	s.sheetHandlers = api.NewSheet(sheetsSvc, workbooksSvc, s.getUserID)
	s.cellHandlers = api.NewCell(cellsSvc, sheetsSvc, workbooksSvc, s.getUserID)
	s.chartsHandlers = api.NewCharts(chartsSvc, sheetsSvc, workbooksSvc, s.getUserID)
	s.importExportHandlers = api.NewImportExport(exportSvc, importSvc, workbooksSvc, sheetsSvc, s.getUserID)
	s.uiHandlers = handler.NewUI(tmpl, usersSvc, workbooksSvc)

	s.setupRoutes()

	// Serve static files
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		c.Writer().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	// Serve uploaded files
	uploadsDir := filepath.Join(cfg.DataDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)
	uploadsHandler := http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir)))
	s.app.Get("/uploads/{path...}", func(c *mizu.Ctx) error {
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		c.Writer().Header().Set("Cache-Control", "public, max-age=86400")
		uploadsHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	slog.Info("Starting Spreadsheet server", "addr", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Service accessors for CLI use
func (s *Server) UserService() users.API         { return s.users }
func (s *Server) WorkbookService() workbooks.API { return s.workbooks }
func (s *Server) SheetService() sheets.API       { return s.sheets }
func (s *Server) CellService() cells.API         { return s.cells }

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler { return s.app }

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// UI routes
	s.app.Get("/", func(c *mizu.Ctx) error {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	})
	s.app.Get("/login", s.uiHandlers.Login)
	s.app.Get("/register", s.uiHandlers.Register)
	s.app.Get("/app", s.uiHandlers.AppRedirect)
	s.app.Get("/s/{workbookID}", s.authRequired(s.uiHandlers.Spreadsheet))
	s.app.Get("/s/{workbookID}/{sheetID}", s.authRequired(s.uiHandlers.Spreadsheet))

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Workbooks
		api.Get("/workbooks", s.authRequired(s.workbookHandlers.List))
		api.Post("/workbooks", s.authRequired(s.workbookHandlers.Create))
		api.Get("/workbooks/{id}", s.authRequired(s.workbookHandlers.Get))
		api.Patch("/workbooks/{id}", s.authRequired(s.workbookHandlers.Update))
		api.Delete("/workbooks/{id}", s.authRequired(s.workbookHandlers.Delete))
		api.Get("/workbooks/{id}/sheets", s.authRequired(s.workbookHandlers.ListSheets))

		// Sheets
		api.Post("/sheets", s.authRequired(s.sheetHandlers.Create))
		api.Get("/sheets/{id}", s.authRequired(s.sheetHandlers.Get))
		api.Patch("/sheets/{id}", s.authRequired(s.sheetHandlers.Update))
		api.Delete("/sheets/{id}", s.authRequired(s.sheetHandlers.Delete))

		// Cells
		api.Get("/sheets/{sheetID}/cells", s.authRequired(s.cellHandlers.GetRange))
		api.Put("/sheets/{sheetID}/cells", s.authRequired(s.cellHandlers.BatchUpdate))
		api.Get("/sheets/{sheetID}/cells/{row}/{col}", s.authRequired(s.cellHandlers.Get))
		api.Put("/sheets/{sheetID}/cells/{row}/{col}", s.authRequired(s.cellHandlers.Set))
		api.Delete("/sheets/{sheetID}/cells/{row}/{col}", s.authRequired(s.cellHandlers.Delete))

		// Cell operations
		api.Post("/sheets/{sheetID}/rows/insert", s.authRequired(s.cellHandlers.InsertRows))
		api.Post("/sheets/{sheetID}/rows/delete", s.authRequired(s.cellHandlers.DeleteRows))
		api.Post("/sheets/{sheetID}/cols/insert", s.authRequired(s.cellHandlers.InsertCols))
		api.Post("/sheets/{sheetID}/cols/delete", s.authRequired(s.cellHandlers.DeleteCols))

		// Merged regions
		api.Get("/sheets/{sheetID}/merges", s.authRequired(s.cellHandlers.GetMerges))
		api.Post("/sheets/{sheetID}/merge", s.authRequired(s.cellHandlers.Merge))
		api.Post("/sheets/{sheetID}/unmerge", s.authRequired(s.cellHandlers.Unmerge))

		// Copy range
		api.Post("/sheets/{sheetID}/copy-range", s.authRequired(s.cellHandlers.CopyRange))

		// Formula evaluation
		api.Post("/formula/evaluate", s.authRequired(s.cellHandlers.Evaluate))

		// Import/Export
		api.Get("/formats", s.authRequired(s.importExportHandlers.SupportedFormats))
		api.Get("/workbooks/{id}/export", s.authRequired(s.importExportHandlers.ExportWorkbook))
		api.Post("/workbooks/{id}/export", s.authRequired(s.importExportHandlers.ExportWorkbook))
		api.Post("/workbooks/{id}/import", s.authRequired(s.importExportHandlers.ImportToWorkbook))
		api.Get("/sheets/{id}/export", s.authRequired(s.importExportHandlers.ExportSheet))
		api.Post("/sheets/{id}/export", s.authRequired(s.importExportHandlers.ExportSheet))
		api.Post("/sheets/{id}/import", s.authRequired(s.importExportHandlers.ImportToSheet))

		// Charts
		api.Post("/charts", s.authRequired(s.chartsHandlers.Create))
		api.Get("/charts/{id}", s.authRequired(s.chartsHandlers.Get))
		api.Patch("/charts/{id}", s.authRequired(s.chartsHandlers.Update))
		api.Delete("/charts/{id}", s.authRequired(s.chartsHandlers.Delete))
		api.Post("/charts/{id}/duplicate", s.authRequired(s.chartsHandlers.Duplicate))
		api.Get("/charts/{id}/data", s.authRequired(s.chartsHandlers.GetData))
		api.Get("/sheets/{sheetId}/charts", s.authRequired(s.chartsHandlers.ListBySheet))
	})
}

// ChartService returns the charts service.
func (s *Server) ChartService() charts.API { return s.charts }

// cellDataProviderAdapter adapts cells.API to charts.CellDataProvider.
type cellDataProviderAdapter struct {
	cells cells.API
}

// GetCellValues retrieves cell values in a range.
func (a *cellDataProviderAdapter) GetCellValues(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([][]interface{}, error) {
	cellList, err := a.cells.GetRange(ctx, sheetID, startRow, startCol, endRow, endCol)
	if err != nil {
		return nil, err
	}

	// Calculate dimensions
	numRows := endRow - startRow + 1
	numCols := endCol - startCol + 1

	// Initialize 2D array
	result := make([][]interface{}, numRows)
	for i := range result {
		result[i] = make([]interface{}, numCols)
	}

	// Populate with cell values
	for _, cell := range cellList {
		rowIdx := cell.Row - startRow
		colIdx := cell.Col - startCol
		if rowIdx >= 0 && rowIdx < numRows && colIdx >= 0 && colIdx < numCols {
			// Use display value if available (for formulas), otherwise use value
			if cell.Formula != "" && cell.Display != "" {
				result[rowIdx][colIdx] = cell.Display
			} else if cell.Value != nil {
				result[rowIdx][colIdx] = cell.Value
			} else {
				result[rowIdx][colIdx] = cell.Display
			}
		}
	}

	return result, nil
}

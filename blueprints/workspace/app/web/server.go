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

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/app/web/handler"
	"github.com/go-mizu/blueprints/workspace/app/web/handler/api"
	"github.com/go-mizu/blueprints/workspace/assets"
	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/comments"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/export"
	"github.com/go-mizu/blueprints/workspace/feature/favorites"
	"github.com/go-mizu/blueprints/workspace/feature/history"
	"github.com/go-mizu/blueprints/workspace/feature/members"
	"github.com/go-mizu/blueprints/workspace/feature/notifications"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/rows"
	"github.com/go-mizu/blueprints/workspace/feature/search"
	"github.com/go-mizu/blueprints/workspace/feature/sharing"
	"github.com/go-mizu/blueprints/workspace/feature/templates"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/feature/views"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
	"github.com/go-mizu/blueprints/workspace/store/duckdb"
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
	users         users.API
	workspaces    workspaces.API
	members       members.API
	pages         pages.API
	blocks        blocks.API
	databases     databases.API
	views         views.API
	rows          rows.API
	comments      comments.API
	sharing       sharing.API
	history       history.API
	notifications notifications.API
	favorites     favorites.API
	search        search.API
	templates     templates.API

	// Export service
	exports *export.Service

	// Handlers
	authHandlers      *api.Auth
	workspaceHandlers *api.Workspace
	pageHandlers      *api.Page
	blockHandlers     *api.Block
	databaseHandlers  *api.Database
	viewHandlers      *api.View
	rowHandlers       *api.Row
	commentHandlers   *api.Comment
	shareHandlers     *api.Share
	favoriteHandlers  *api.Favorite
	searchHandlers    *api.Search
	mediaHandlers     *api.Media
	exportHandlers    *api.Export
	uiHandlers        *handler.UI
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "workspace.duckdb")
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
	workspacesStore := duckdb.NewWorkspacesStore(db)
	membersStore := duckdb.NewMembersStore(db)
	pagesStore := duckdb.NewPagesStore(db)
	blocksStore := duckdb.NewBlocksStore(db)
	databasesStore := duckdb.NewDatabasesStore(db)
	viewsStore := duckdb.NewViewsStore(db)
	rowsStore := duckdb.NewRowsStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	sharesStore := duckdb.NewSharesStore(db)
	historyStore := duckdb.NewHistoryStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)
	favoritesStore := duckdb.NewFavoritesStore(db)
	templatesStore := duckdb.NewTemplatesStore(db)

	// Create services
	usersSvc := users.NewService(usersStore)
	workspacesSvc := workspaces.NewService(workspacesStore)
	membersSvc := members.NewService(membersStore, usersSvc)
	pagesSvc := pages.NewService(pagesStore, usersSvc)
	blocksSvc := blocks.NewService(blocksStore)
	databasesSvc := databases.NewService(databasesStore)
	viewsSvc := views.NewService(viewsStore, pagesSvc)
	rowsSvc := rows.NewService(rowsStore, pagesSvc) // Rows now use pages service
	commentsSvc := comments.NewService(commentsStore, usersSvc)
	sharingSvc := sharing.NewService(sharesStore, usersSvc)
	historySvc := history.NewService(historyStore, usersSvc, pagesSvc, blocksSvc)
	notificationsSvc := notifications.NewService(notificationsStore, usersSvc, pagesSvc)
	favoritesSvc := favorites.NewService(favoritesStore, pagesSvc)
	templatesSvc := templates.NewService(templatesStore, pagesSvc)
	searchSvc := search.NewService(duckdb.NewSearchStore(db), pagesSvc, databasesSvc)
	exportSvc := export.NewService(pagesSvc, blocksSvc, databasesSvc, filepath.Join(cfg.DataDir, "exports"))

	s := &Server{
		app:           mizu.New(),
		cfg:           cfg,
		db:            db,
		users:         usersSvc,
		workspaces:    workspacesSvc,
		members:       membersSvc,
		pages:         pagesSvc,
		blocks:        blocksSvc,
		databases:     databasesSvc,
		views:         viewsSvc,
		rows:          rowsSvc,
		comments:      commentsSvc,
		sharing:       sharingSvc,
		history:       historySvc,
		notifications: notificationsSvc,
		favorites:     favoritesSvc,
		search:        searchSvc,
		templates:     templatesSvc,
		exports:       exportSvc,
	}

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	// Create handlers
	s.authHandlers = api.NewAuth(usersSvc)
	s.workspaceHandlers = api.NewWorkspace(workspacesSvc, membersSvc, s.getUserID)
	s.pageHandlers = api.NewPage(pagesSvc, blocksSvc, s.getUserID)
	s.blockHandlers = api.NewBlock(blocksSvc, s.getUserID)
	s.databaseHandlers = api.NewDatabase(databasesSvc, s.getUserID)
	s.viewHandlers = api.NewView(viewsSvc, s.getUserID)
	s.rowHandlers = api.NewRow(rowsSvc, blocksSvc, commentsSvc, databasesSvc, s.getUserID)
	s.commentHandlers = api.NewComment(commentsSvc, s.getUserID)
	s.shareHandlers = api.NewShare(sharingSvc, s.getUserID)
	s.favoriteHandlers = api.NewFavorite(favoritesSvc, s.getUserID)
	s.searchHandlers = api.NewSearch(searchSvc, s.getUserID)
	s.mediaHandlers = api.NewMedia(filepath.Join(cfg.DataDir, "uploads"), s.getUserID)
	s.exportHandlers = api.NewExport(exportSvc, s.getUserID)
	s.uiHandlers = handler.NewUI(tmpl, usersSvc, workspacesSvc, membersSvc, pagesSvc, blocksSvc, databasesSvc, viewsSvc, rowsSvc, favoritesSvc, s.getUserID)

	s.setupRoutes()

	// Seed dev data if in dev mode
	if isDevMode() {
		if err := s.seedDevData(); err != nil {
			slog.Warn("failed to seed dev data", "error", err)
		}
	}

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
	slog.Info("Starting Workspace server", "addr", s.cfg.Addr)
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
func (s *Server) UserService() users.API           { return s.users }
func (s *Server) WorkspaceService() workspaces.API { return s.workspaces }
func (s *Server) MemberService() members.API       { return s.members }
func (s *Server) PageService() pages.API           { return s.pages }
func (s *Server) BlockService() blocks.API         { return s.blocks }
func (s *Server) DatabaseService() databases.API   { return s.databases }
func (s *Server) ViewService() views.API           { return s.views }
func (s *Server) RowService() rows.API             { return s.rows }
func (s *Server) CommentService() comments.API     { return s.comments }

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler { return s.app }

// seedDevData creates test data for development mode.
func (s *Server) seedDevData() error {
	ctx := context.Background()

	// Check if dev database already exists
	_, err := s.databases.GetByID(ctx, "dev-db-001")
	if err == nil {
		slog.Info("Dev data already exists, skipping seed")
		return nil
	}

	slog.Info("Seeding dev data...")

	// Create dev page for export testing (check if it exists)
	_, devPageErr := s.pages.GetByID(ctx, "dev-page-001")
	if devPageErr != nil {
		// Create a standalone dev page for testing export
		devPage, err := s.pages.Create(ctx, &pages.CreateIn{
			WorkspaceID: "dev-ws-001",
			ParentType:  pages.ParentWorkspace,
			ParentID:    "dev-ws-001",
			Title:       "Development Test Page",
			Icon:        "📄",
			CreatedBy:   devUserID,
		})
		if err == nil {
			// Override ID for consistency
			s.db.ExecContext(ctx, "UPDATE pages SET id = ? WHERE id = ?", "dev-page-001", devPage.ID)

			// Add some sample blocks to the dev page
			sampleDevBlocks := []struct {
				typ   string
				cont  blocks.Content
				order int
			}{
				{string(blocks.BlockHeading1), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Welcome to the Test Page"}}}, 0},
				{string(blocks.BlockParagraph), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "This is a development test page used for testing the export functionality."}}}, 1},
				{string(blocks.BlockHeading2), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Features"}}}, 2},
				{string(blocks.BlockBulletList), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Export to PDF"}}}, 3},
				{string(blocks.BlockBulletList), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Export to HTML"}}}, 4},
				{string(blocks.BlockBulletList), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Export to Markdown"}}}, 5},
				{string(blocks.BlockDivider), blocks.Content{}, 6},
				{string(blocks.BlockCallout), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "PDF export requires Chrome/Chromium to be installed."}}, Icon: "💡", Color: "yellow"}, 7},
			}

			for _, b := range sampleDevBlocks {
				s.blocks.Create(ctx, &blocks.CreateIn{
					PageID:    "dev-page-001",
					Type:      blocks.BlockType(b.typ),
					Content:   b.cont,
					Position:  b.order,
					CreatedBy: devUserID,
				})
			}
			slog.Info("Created dev-page-001 for export testing")
		}
	}

	// Create dev workspace first (if it doesn't exist)
	ws, _ := s.workspaces.GetByID(ctx, "dev-ws-001")
	if ws == nil {
		ws, err = s.workspaces.Create(ctx, devUserID, &workspaces.CreateIn{
			Name: "Dev Workspace",
			Slug: "dev",
		})
		if err != nil {
			return fmt.Errorf("create dev workspace: %w", err)
		}
		// Override ID for consistency
		s.db.ExecContext(ctx, "UPDATE workspaces SET id = ? WHERE id = ?", "dev-ws-001", ws.ID)
		ws.ID = "dev-ws-001"
	}

	// Create Tasks Tracker database (like Notion screenshot)
	db, err := s.databases.Create(ctx, &databases.CreateIn{
		WorkspaceID: ws.ID,
		Title:       "Tasks Tracker",
		Properties: []databases.Property{
			{ID: "title", Name: "Task name", Type: databases.PropTitle},
			{ID: "status", Name: "Status", Type: databases.PropStatus, Config: databases.StatusConfig{
				Options: []databases.StatusOption{
					{ID: "not-started", Name: "Not started", Color: "gray"},
					{ID: "in-progress", Name: "In progress", Color: "blue"},
					{ID: "done", Name: "Done", Color: "green"},
				},
			}},
			{ID: "assignee", Name: "Assignee", Type: databases.PropPerson},
			{ID: "due_date", Name: "Due date", Type: databases.PropDate},
			{ID: "priority", Name: "Priority", Type: databases.PropSelect, Config: databases.SelectConfig{
				Options: []databases.SelectOption{
					{ID: "low", Name: "Low", Color: "gray"},
					{ID: "medium", Name: "Medium", Color: "yellow"},
					{ID: "high", Name: "High", Color: "red"},
					{ID: "urgent", Name: "Urgent", Color: "pink"},
				},
			}},
			{ID: "tags", Name: "Tags", Type: databases.PropMultiSelect, Config: databases.SelectConfig{
				Options: []databases.SelectOption{
					{ID: "website", Name: "Website", Color: "blue"},
					{ID: "help-center", Name: "Help Center", Color: "green"},
					{ID: "release", Name: "Release", Color: "purple"},
					{ID: "marketing", Name: "Marketing", Color: "orange"},
					{ID: "documentation", Name: "Documentation", Color: "yellow"},
				},
			}},
			{ID: "progress", Name: "Progress", Type: databases.PropNumber},
			{ID: "description", Name: "Description", Type: databases.PropRichText},
			{ID: "url", Name: "URL", Type: databases.PropURL},
			{ID: "email", Name: "Email", Type: databases.PropEmail},
		},
		CreatedBy: devUserID,
	})
	if err != nil {
		return fmt.Errorf("create dev database: %w", err)
	}

	// Override database ID for consistency
	s.db.ExecContext(ctx, "UPDATE databases SET id = ? WHERE id = ?", "dev-db-001", db.ID)

	// Create realistic task rows (like Notion Tasks Tracker)
	// Rows are now pages with database_id set
	devRows := []struct {
		id    string
		props map[string]interface{}
	}{
		{
			id: "row1",
			props: map[string]interface{}{
				"title":       "Improve website copy",
				"description": "Review and update all marketing copy on the main website. Focus on clarity and conversion optimization.",
				"progress":    100,
				"status":      "done",
				"tags":        []string{"website", "marketing"},
				"priority":    "high",
				"due_date":    "2025-02-03",
				"assignee":    "Sarah Chen",
			},
		},
		{
			id: "row2",
			props: map[string]interface{}{
				"title":       "Update help center & FAQ",
				"description": "Add new articles covering recent feature releases. Update existing documentation for accuracy.",
				"progress":    60,
				"status":      "in-progress",
				"tags":        []string{"help-center", "documentation"},
				"priority":    "medium",
				"due_date":    "2025-02-20",
				"assignee":    "Mike Johnson",
			},
		},
		{
			id: "row3",
			props: map[string]interface{}{
				"title":       "Publish release notes@",
				"description": "Draft and publish release notes for version 2.5. Include all new features and bug fixes.",
				"progress":    0,
				"status":      "not-started",
				"tags":        []string{"release"},
				"priority":    "high",
				"due_date":    "2025-02-28",
				"assignee":    "",
			},
		},
		{
			id: "row4",
			props: map[string]interface{}{
				"title":       "Design new onboarding flow",
				"description": "Create wireframes and mockups for the new user onboarding experience. Focus on reducing time-to-value.",
				"progress":    35,
				"status":      "in-progress",
				"tags":        []string{"website"},
				"priority":    "urgent",
				"due_date":    "2025-01-02",
				"assignee":    "Emily Davis",
			},
		},
		{
			id: "row5",
			props: map[string]interface{}{
				"title":       "Prepare Q1 marketing campaign",
				"description": "Develop comprehensive marketing plan including social media, email campaigns, and paid advertising.",
				"progress":    0,
				"status":      "not-started",
				"tags":        []string{"marketing"},
				"priority":    "medium",
				"due_date":    "2024-12-31",
				"assignee":    "Alex Thompson",
			},
		},
		{
			id: "row6",
			props: map[string]interface{}{
				"title":       "Review API documentation",
				"description": "Audit current API docs for completeness and accuracy. Add missing endpoints and examples.",
				"progress":    0,
				"status":      "not-started",
				"tags":        []string{"documentation"},
				"priority":    "low",
				"due_date":    "2024-12-31",
				"assignee":    "Chris Wilson",
			},
		},
		{
			id: "row7",
			props: map[string]interface{}{
				"title":       "Customer feedback analysis",
				"description": "Analyze recent customer feedback and support tickets. Identify top feature requests and pain points.",
				"progress":    80,
				"status":      "in-progress",
				"tags":        []string{"help-center"},
				"priority":    "high",
				"due_date":    "2025-01-15",
				"assignee":    "Jordan Lee",
			},
		},
		{
			id: "row8",
			props: map[string]interface{}{
				"title":       "Launch blog series",
				"description": "Write and publish a 5-part blog series on best practices for productivity. Include case studies.",
				"progress":    45,
				"status":      "in-progress",
				"tags":        []string{"marketing", "website"},
				"priority":    "medium",
				"due_date":    "2025-02-10",
				"assignee":    "Taylor Brown",
			},
		},
	}

	for _, r := range devRows {
		row, err := s.rows.Create(ctx, &rows.CreateIn{
			DatabaseID:  "dev-db-001",
			WorkspaceID: ws.ID,
			Properties:  r.props,
			CreatedBy:   devUserID,
		})
		if err != nil {
			slog.Warn("failed to create dev row", "rowID", r.id, "error", err)
			continue
		}
		// Override row ID for consistency
		s.db.ExecContext(ctx, "UPDATE pages SET id = ? WHERE id = ?", r.id, row.ID)
	}

	// Create a default view
	_, err = s.views.Create(ctx, &views.CreateIn{
		DatabaseID: "dev-db-001",
		Name:       "Table view",
		Type:       "table",
		CreatedBy:  devUserID,
	})
	if err != nil {
		slog.Warn("failed to create dev view", "error", err)
	}

	// Seed sample row comments using polymorphic comments
	sampleComments := []struct {
		rowID   string
		content string
	}{
		{"row1", "Great progress on this! The copy looks much better now."},
		{"row1", "Agreed, the conversion rates should improve with these changes."},
		{"row2", "We need to prioritize the FAQ section - lots of customer questions coming in."},
		{"row3", "Please include screenshots for all new features."},
	}

	for _, c := range sampleComments {
		_, err := s.comments.Create(ctx, &comments.CreateIn{
			WorkspaceID: ws.ID,
			TargetType:  comments.TargetDatabaseRow,
			TargetID:    c.rowID,
			Content:     []blocks.RichText{{Type: "text", Text: c.content}},
			AuthorID:    devUserID,
		})
		if err != nil {
			slog.Warn("failed to create sample comment", "error", err)
		}
	}

	// Seed sample content blocks for row3 (which is a page)
	// Using the standard blocks service with page_id = row3
	sampleBlocks := []struct {
		typ   string
		cont  blocks.Content
		order int
	}{
		{string(blocks.BlockHeading2), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Task description"}}}, 0},
		{string(blocks.BlockParagraph), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Provide an overview of the task and related details."}}}, 1},
		{string(blocks.BlockHeading2), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Sub-tasks"}}}, 2},
		{string(blocks.BlockTodo), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Review previous release notes format"}}, Checked: ptrBool(false)}, 3},
		{string(blocks.BlockTodo), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Gather feature list from engineering"}}, Checked: ptrBool(false)}, 4},
		{string(blocks.BlockTodo), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Write initial draft"}}, Checked: ptrBool(false)}, 5},
		{string(blocks.BlockTodo), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Get approval from PM"}}, Checked: ptrBool(false)}, 6},
		{string(blocks.BlockTodo), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Publish to blog and email"}}, Checked: ptrBool(false)}, 7},
		{string(blocks.BlockHeading2), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Supporting files"}}}, 8},
		{string(blocks.BlockCallout), blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Add any relevant documents, designs, or references here."}}, Icon: "📎"}, 9},
	}

	for _, b := range sampleBlocks {
		_, err := s.blocks.Create(ctx, &blocks.CreateIn{
			PageID:    "row3", // row3 is a page
			Type:      blocks.BlockType(b.typ),
			Content:   b.cont,
			Position:  b.order,
			CreatedBy: devUserID,
		})
		if err != nil {
			slog.Warn("failed to create sample block", "error", err)
		}
	}

	slog.Info("Dev data seeded successfully")
	return nil
}

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
	s.app.Get("/w/{workspace}", s.authRequired(s.uiHandlers.Workspace))
	s.app.Get("/w/{workspace}/p/{pageID}", s.authRequired(s.uiHandlers.Page))
	s.app.Get("/w/{workspace}/d/{databaseID}", s.authRequired(s.uiHandlers.Database))
	s.app.Get("/w/{workspace}/search", s.authRequired(s.uiHandlers.Search))
	s.app.Get("/w/{workspace}/settings", s.authRequired(s.uiHandlers.Settings))

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Workspaces
		api.Get("/workspaces", s.authRequired(s.workspaceHandlers.List))
		api.Post("/workspaces", s.authRequired(s.workspaceHandlers.Create))
		api.Get("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Get))
		api.Patch("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Update))
		api.Delete("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Delete))
		api.Get("/workspaces/{id}/members", s.authRequired(s.workspaceHandlers.ListMembers))
		api.Post("/workspaces/{id}/members", s.authRequired(s.workspaceHandlers.AddMember))

		// Pages
		api.Post("/pages", s.authRequired(s.pageHandlers.Create))
		api.Get("/pages/{id}", s.authRequired(s.pageHandlers.Get))
		api.Patch("/pages/{id}", s.authRequired(s.pageHandlers.Update))
		api.Delete("/pages/{id}", s.authRequired(s.pageHandlers.Delete))
		api.Get("/workspaces/{workspaceID}/pages", s.authRequired(s.pageHandlers.List))
		api.Get("/pages/{id}/blocks", s.authRequired(s.pageHandlers.GetBlocks))
		api.Put("/pages/{id}/blocks", s.authRequired(s.pageHandlers.UpdateBlocks))
		api.Post("/pages/{id}/archive", s.authRequired(s.pageHandlers.Archive))
		api.Post("/pages/{id}/restore", s.authRequired(s.pageHandlers.Restore))
		api.Post("/pages/{id}/duplicate", s.authRequired(s.pageHandlers.Duplicate))

		// Blocks
		api.Post("/blocks", s.authRequired(s.blockHandlers.Create))
		api.Patch("/blocks/{id}", s.authRequired(s.blockHandlers.Update))
		api.Delete("/blocks/{id}", s.authRequired(s.blockHandlers.Delete))
		api.Post("/blocks/{id}/move", s.authRequired(s.blockHandlers.Move))

		// Databases
		api.Post("/databases", s.authRequired(s.databaseHandlers.Create))
		api.Get("/databases/{id}", s.authRequired(s.databaseHandlers.Get))
		api.Patch("/databases/{id}", s.authRequired(s.databaseHandlers.Update))
		api.Delete("/databases/{id}", s.authRequired(s.databaseHandlers.Delete))
		api.Post("/databases/{id}/properties", s.authRequired(s.databaseHandlers.AddProperty))
		api.Patch("/databases/{id}/properties/{propID}", s.authRequired(s.databaseHandlers.UpdateProperty))
		api.Delete("/databases/{id}/properties/{propID}", s.authRequired(s.databaseHandlers.DeleteProperty))

		// Views
		api.Post("/views", s.authRequired(s.viewHandlers.Create))
		api.Get("/views/{id}", s.authRequired(s.viewHandlers.Get))
		api.Patch("/views/{id}", s.authRequired(s.viewHandlers.Update))
		api.Put("/views/{id}", s.authRequired(s.viewHandlers.Update))
		api.Delete("/views/{id}", s.authRequired(s.viewHandlers.Delete))
		api.Get("/databases/{id}/views", s.authRequired(s.viewHandlers.List))
		api.Post("/databases/{id}/views", s.authRequired(s.viewHandlers.Create))
		api.Post("/views/{id}/query", s.authRequired(s.viewHandlers.Query))

		// Rows (database rows are pages with database_id set)
		api.Get("/databases/{id}/rows", s.authRequired(s.rowHandlers.List))
		api.Post("/databases/{id}/rows", s.authRequired(s.rowHandlers.Create))
		api.Get("/rows/{id}", s.authRequired(s.rowHandlers.Get))
		api.Patch("/rows/{id}", s.authRequired(s.rowHandlers.Update))
		api.Delete("/rows/{id}", s.authRequired(s.rowHandlers.Delete))
		api.Post("/rows/{id}/duplicate", s.authRequired(s.rowHandlers.Duplicate))

		// Row Comments (using unified comments with target_type=database_row)
		api.Get("/rows/{id}/comments", s.authRequired(s.rowHandlers.ListComments))
		api.Post("/rows/{id}/comments", s.authRequired(s.rowHandlers.CreateComment))
		api.Patch("/row-comments/{id}", s.authRequired(s.rowHandlers.UpdateComment))
		api.Delete("/row-comments/{id}", s.authRequired(s.rowHandlers.DeleteComment))
		api.Post("/row-comments/{id}/resolve", s.authRequired(s.rowHandlers.ResolveComment))
		api.Post("/row-comments/{id}/unresolve", s.authRequired(s.rowHandlers.UnresolveComment))

		// Row Content Blocks (using unified blocks with page_id = row id)
		api.Get("/rows/{id}/blocks", s.authRequired(s.rowHandlers.ListBlocks))
		api.Post("/rows/{id}/blocks", s.authRequired(s.rowHandlers.CreateBlock))
		api.Post("/rows/{id}/blocks/reorder", s.authRequired(s.rowHandlers.ReorderBlocks))
		api.Patch("/row-blocks/{id}", s.authRequired(s.rowHandlers.UpdateBlock))
		api.Delete("/row-blocks/{id}", s.authRequired(s.rowHandlers.DeleteBlock))

		// Comments
		api.Post("/comments", s.authRequired(s.commentHandlers.Create))
		api.Patch("/comments/{id}", s.authRequired(s.commentHandlers.Update))
		api.Delete("/comments/{id}", s.authRequired(s.commentHandlers.Delete))
		api.Get("/pages/{id}/comments", s.authRequired(s.commentHandlers.List))
		api.Post("/comments/{id}/resolve", s.authRequired(s.commentHandlers.Resolve))

		// Sharing
		api.Post("/pages/{id}/shares", s.authRequired(s.shareHandlers.Create))
		api.Get("/pages/{id}/shares", s.authRequired(s.shareHandlers.List))
		api.Delete("/shares/{id}", s.authRequired(s.shareHandlers.Delete))

		// Favorites
		api.Post("/favorites", s.authRequired(s.favoriteHandlers.Add))
		api.Delete("/favorites/{pageID}", s.authRequired(s.favoriteHandlers.Remove))
		api.Get("/workspaces/{id}/favorites", s.authRequired(s.favoriteHandlers.List))

		// Search
		api.Get("/workspaces/{id}/search", s.authRequired(s.searchHandlers.Search))
		api.Get("/workspaces/{id}/quick-search", s.authRequired(s.searchHandlers.QuickSearch))
		api.Get("/workspaces/{id}/recent", s.authRequired(s.searchHandlers.Recent))

		// Media
		api.Post("/media/upload", s.authRequired(s.mediaHandlers.Upload))

		// Export
		api.Post("/pages/{id}/export", s.authRequired(s.exportHandlers.ExportPage))
		api.Get("/exports/{id}", s.authRequired(s.exportHandlers.GetExport))
		api.Get("/exports/{id}/download", s.authRequired(s.exportHandlers.Download))
	})
}

func ptrBool(b bool) *bool {
	return &b
}

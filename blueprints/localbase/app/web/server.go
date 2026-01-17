package web

import (
	"io/fs"
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/middleware"
	"github.com/go-mizu/mizu/blueprints/localbase/assets"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// NewServer creates a new HTTP server with all routes configured.
func NewServer(store *postgres.Store, devMode bool) (http.Handler, error) {
	app := mizu.New()

	// Create handlers
	authHandler := api.NewAuthHandler(store)
	storageHandler := api.NewStorageHandler(store)
	databaseHandler := api.NewDatabaseHandler(store)
	functionsHandler := api.NewFunctionsHandler(store)
	realtimeHandler := api.NewRealtimeHandler(store)
	dashboardHandler := api.NewDashboardHandler(store)
	pgmetaHandler := api.NewPGMetaHandler(store)
	logsHandler := api.NewLogsHandler()
	settingsHandler := api.NewSettingsHandler(store)

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "healthy"})
	})

	// API key middleware for Supabase compatibility (defined early for reuse)
	apiKeyMw := middleware.APIKey(middleware.DefaultAPIKeyConfig())
	serviceRoleMw := middleware.RequireServiceRole()
	authRateLimitMw := middleware.RateLimit(middleware.AuthRateLimitConfig())

	// Auth API (GoTrue compatible)
	app.Group("/auth/v1", func(auth *mizu.Router) {
		// Apply API key middleware to all auth endpoints
		auth.Use(apiKeyMw)
		// Apply rate limiting to auth endpoints for brute force protection
		auth.Use(authRateLimitMw)

		// Public auth endpoints
		auth.Post("/signup", authHandler.Signup)
		auth.Post("/token", authHandler.Token)
		auth.Post("/logout", authHandler.Logout)
		auth.Post("/recover", authHandler.Recover)
		auth.Get("/user", authHandler.GetUser)
		auth.Put("/user", authHandler.UpdateUser)
		auth.Post("/otp", authHandler.SendOTP)
		auth.Post("/verify", authHandler.Verify)
		auth.Post("/factors", authHandler.EnrollMFA)
		auth.Delete("/factors/{id}", authHandler.UnenrollMFA)
		auth.Post("/factors/{id}/challenge", authHandler.ChallengeMFA)
		auth.Post("/factors/{id}/verify", authHandler.VerifyMFA)

		// Admin endpoints (require service_role)
		auth.Group("/admin", func(admin *mizu.Router) {
			admin.Use(serviceRoleMw)
			admin.Get("/users", authHandler.ListUsers)
			admin.Post("/users", authHandler.CreateUser)
			admin.Get("/users/{id}", authHandler.GetUserByID)
			admin.Put("/users/{id}", authHandler.UpdateUserByID)
			admin.Delete("/users/{id}", authHandler.DeleteUser)
		})
	})

	// Storage API (Supabase Storage compatible)
	// Apply API key middleware for Storage endpoints
	app.Group("/storage/v1", func(storage *mizu.Router) {
		storage.Use(apiKeyMw)
		// Bucket endpoints
		storage.Get("/bucket", storageHandler.ListBuckets)
		storage.Post("/bucket", storageHandler.CreateBucket)
		storage.Get("/bucket/{id}", storageHandler.GetBucket)
		storage.Put("/bucket/{id}", storageHandler.UpdateBucket)
		storage.Delete("/bucket/{id}", storageHandler.DeleteBucket)
		storage.Post("/bucket/{id}/empty", storageHandler.EmptyBucket)

		// Object operations
		storage.Post("/object/list/{bucket}", storageHandler.ListObjects)
		storage.Post("/object/move", storageHandler.MoveObject)
		storage.Post("/object/copy", storageHandler.CopyObject)

		// Signed URLs
		storage.Post("/object/sign/{bucket}/{path...}", storageHandler.CreateSignedURL)
		storage.Post("/object/sign/{bucket}", storageHandler.CreateSignedURLs)
		storage.Post("/object/upload/sign/{bucket}/{path...}", storageHandler.CreateUploadSignedURL)

		// Object CRUD with path
		storage.Post("/object/{bucket}/{path...}", storageHandler.UploadObject)
		storage.Get("/object/{bucket}/{path...}", storageHandler.DownloadObject)
		storage.Put("/object/{bucket}/{path...}", storageHandler.UpdateObject)
		storage.Delete("/object/{bucket}/{path...}", storageHandler.DeleteObject)
		storage.Delete("/object/{bucket}", storageHandler.DeleteObjects)

		// Public access
		storage.Get("/object/public/{bucket}/{path...}", storageHandler.DownloadPublicObject)
		storage.Get("/object/info/public/{bucket}/{path...}", storageHandler.GetPublicObjectInfo)

		// Authenticated access
		storage.Get("/object/authenticated/{bucket}/{path...}", storageHandler.DownloadAuthenticatedObject)
		storage.Get("/object/info/authenticated/{bucket}/{path...}", storageHandler.GetAuthenticatedObjectInfo)

		// Object info
		storage.Get("/object/info/{bucket}/{path...}", storageHandler.GetObjectInfo)
	})

	// REST API (PostgREST compatible)
	app.Group("/rest/v1", func(rest *mizu.Router) {
		rest.Use(apiKeyMw)
		rest.Get("/{table}", databaseHandler.SelectTable)
		rest.Head("/{table}", databaseHandler.SelectTable) // Support HEAD for count
		rest.Post("/{table}", databaseHandler.InsertTable)
		rest.Patch("/{table}", databaseHandler.UpdateTable)
		rest.Delete("/{table}", databaseHandler.DeleteTable)
		rest.Post("/rpc/{function}", databaseHandler.CallFunction)
		rest.Get("/rpc/{function}", databaseHandler.CallFunction) // Support GET for RPC
	})

	// Database API (Dashboard) - Requires service_role for all operations
	app.Group("/api/database", func(database *mizu.Router) {
		database.Use(apiKeyMw)
		database.Use(serviceRoleMw)
		database.Get("/tables", databaseHandler.ListTables)
		database.Post("/tables", databaseHandler.CreateTable)
		database.Get("/tables/{schema}/{name}", databaseHandler.GetTable)
		database.Delete("/tables/{schema}/{name}", databaseHandler.DropTable)

		database.Get("/tables/{schema}/{name}/columns", databaseHandler.ListColumns)
		database.Post("/tables/{schema}/{name}/columns", databaseHandler.AddColumn)
		database.Put("/tables/{schema}/{name}/columns/{column}", databaseHandler.AlterColumn)
		database.Delete("/tables/{schema}/{name}/columns/{column}", databaseHandler.DropColumn)

		database.Get("/schemas", databaseHandler.ListSchemas)
		database.Post("/schemas", databaseHandler.CreateSchema)

		database.Get("/extensions", databaseHandler.ListExtensions)
		database.Post("/extensions", databaseHandler.EnableExtension)

		database.Get("/policies/{schema}/{table}", databaseHandler.ListPolicies)
		database.Post("/policies", databaseHandler.CreatePolicy)
		database.Delete("/policies/{schema}/{table}/{name}", databaseHandler.DropPolicy)

		database.Post("/query", databaseHandler.ExecuteQuery)
	})

	// Functions API - Requires service_role for management operations
	app.Group("/api/functions", func(functions *mizu.Router) {
		functions.Use(apiKeyMw)
		functions.Use(serviceRoleMw)
		functions.Get("", functionsHandler.ListFunctions)
		functions.Post("", functionsHandler.CreateFunction)
		functions.Get("/{id}", functionsHandler.GetFunction)
		functions.Put("/{id}", functionsHandler.UpdateFunction)
		functions.Delete("/{id}", functionsHandler.DeleteFunction)
		functions.Post("/{id}/deploy", functionsHandler.DeployFunction)
		functions.Get("/{id}/deployments", functionsHandler.ListDeployments)

		functions.Get("/secrets", functionsHandler.ListSecrets)
		functions.Post("/secrets", functionsHandler.CreateSecret)
		functions.Delete("/secrets/{name}", functionsHandler.DeleteSecret)
	})

	// Public function invocation (Supabase-compatible: supports all HTTP methods)
	app.Group("/functions/v1", func(functions *mizu.Router) {
		// Apply API key middleware (optional mode for functions that don't require JWT)
		optionalAPIKeyMw := middleware.APIKey(&middleware.APIKeyConfig{
			JWTSecret:         middleware.DefaultAPIKeyConfig().JWTSecret,
			ValidateSignature: middleware.DefaultAPIKeyConfig().ValidateSignature,
			Optional:          true, // Allow requests without auth for functions with verify_jwt=false
			AnonKey:           middleware.DefaultAPIKeyConfig().AnonKey,
			ServiceKey:        middleware.DefaultAPIKeyConfig().ServiceKey,
		})
		functions.Use(optionalAPIKeyMw)

		// Support all HTTP methods for function invocation
		functions.Get("/{name}", functionsHandler.InvokeFunction)
		functions.Post("/{name}", functionsHandler.InvokeFunction)
		functions.Put("/{name}", functionsHandler.InvokeFunction)
		functions.Patch("/{name}", functionsHandler.InvokeFunction)
		functions.Delete("/{name}", functionsHandler.InvokeFunction)
		// OPTIONS for CORS preflight
		functions.Handle("OPTIONS", "/{name}", functionsHandler.InvokeFunctionOptions)

		// Support subpath routing: /functions/v1/{name}/{subpath...}
		functions.Get("/{name}/{subpath...}", functionsHandler.InvokeFunctionWithPath)
		functions.Post("/{name}/{subpath...}", functionsHandler.InvokeFunctionWithPath)
		functions.Put("/{name}/{subpath...}", functionsHandler.InvokeFunctionWithPath)
		functions.Patch("/{name}/{subpath...}", functionsHandler.InvokeFunctionWithPath)
		functions.Delete("/{name}/{subpath...}", functionsHandler.InvokeFunctionWithPath)
		functions.Handle("OPTIONS", "/{name}/{subpath...}", functionsHandler.InvokeFunctionOptions)
	})

	// Realtime API - Requires authentication
	app.Group("/api/realtime", func(realtime *mizu.Router) {
		realtime.Use(apiKeyMw)
		realtime.Use(serviceRoleMw)
		realtime.Get("/channels", realtimeHandler.ListChannels)
		realtime.Get("/stats", realtimeHandler.GetStats)
	})

	// WebSocket endpoint for realtime
	app.Get("/realtime/v1/websocket", realtimeHandler.WebSocket)

	// Dashboard API - Requires authentication (SEC-018 fix)
	app.Group("/api", func(dashboard *mizu.Router) {
		dashboard.Use(apiKeyMw)
		dashboard.Use(serviceRoleMw)
		dashboard.Get("/dashboard/stats", dashboardHandler.GetStats)
		dashboard.Get("/dashboard/health", dashboardHandler.GetHealth)
	})

	// Logs Explorer API - Supabase Dashboard compatibility
	app.Group("/api/logs", func(logs *mizu.Router) {
		logs.Use(apiKeyMw)
		logs.Use(serviceRoleMw)
		logs.Get("", logsHandler.ListLogs)
		logs.Get("/types", logsHandler.ListLogTypes)
		logs.Post("/search", logsHandler.SearchLogs)
		logs.Get("/export", logsHandler.ExportLogs)
	})

	// Settings API - Supabase Dashboard compatibility
	app.Group("/api/settings", func(settings *mizu.Router) {
		settings.Use(apiKeyMw)
		settings.Use(serviceRoleMw)
		settings.Get("", settingsHandler.GetAllSettings)
		settings.Get("/project", settingsHandler.GetProjectSettings)
		settings.Patch("/project", settingsHandler.UpdateProjectSettings)
		settings.Get("/api", settingsHandler.GetAPISettings)
		settings.Patch("/api", settingsHandler.UpdateAPISettings)
		settings.Get("/auth", settingsHandler.GetAuthSettings)
		settings.Patch("/auth", settingsHandler.UpdateAuthSettings)
		settings.Get("/database", settingsHandler.GetDatabaseSettings)
		settings.Patch("/database", settingsHandler.UpdateDatabaseSettings)
		settings.Get("/storage", settingsHandler.GetStorageSettings)
		settings.Patch("/storage", settingsHandler.UpdateStorageSettings)
	})

	// postgres-meta API - Supabase Dashboard compatibility
	app.Group("/api/pg", func(pg *mizu.Router) {
		pg.Use(apiKeyMw)
		pg.Use(serviceRoleMw)

		// Config
		pg.Get("/config/version", pgmetaHandler.GetVersion)

		// Indexes
		pg.Get("/indexes", pgmetaHandler.ListIndexes)
		pg.Post("/indexes", pgmetaHandler.CreateIndex)
		pg.Delete("/indexes/{id}", pgmetaHandler.DropIndex)

		// Views
		pg.Get("/views", pgmetaHandler.ListViews)
		pg.Post("/views", pgmetaHandler.CreateView)
		pg.Patch("/views/{id}", pgmetaHandler.UpdateView)
		pg.Delete("/views/{id}", pgmetaHandler.DropView)

		// Materialized Views
		pg.Get("/materialized-views", pgmetaHandler.ListMaterializedViews)
		pg.Post("/materialized-views", pgmetaHandler.CreateMaterializedView)
		pg.Post("/materialized-views/{id}/refresh", pgmetaHandler.RefreshMaterializedView)
		pg.Delete("/materialized-views/{id}", pgmetaHandler.DropMaterializedView)

		// Foreign Tables
		pg.Get("/foreign-tables", pgmetaHandler.ListForeignTables)

		// Triggers
		pg.Get("/triggers", pgmetaHandler.ListTriggers)
		pg.Post("/triggers", pgmetaHandler.CreateTrigger)
		pg.Delete("/triggers/{id}", pgmetaHandler.DropTrigger)

		// Types
		pg.Get("/types", pgmetaHandler.ListTypes)
		pg.Post("/types", pgmetaHandler.CreateType)
		pg.Delete("/types/{id}", pgmetaHandler.DropType)

		// Roles
		pg.Get("/roles", pgmetaHandler.ListRoles)
		pg.Post("/roles", pgmetaHandler.CreateRole)
		pg.Patch("/roles/{id}", pgmetaHandler.UpdateRole)
		pg.Delete("/roles/{id}", pgmetaHandler.DropRole)

		// Publications
		pg.Get("/publications", pgmetaHandler.ListPublications)
		pg.Post("/publications", pgmetaHandler.CreatePublication)
		pg.Delete("/publications/{id}", pgmetaHandler.DropPublication)

		// Privileges
		pg.Get("/table-privileges", pgmetaHandler.ListTablePrivileges)
		pg.Get("/column-privileges", pgmetaHandler.ListColumnPrivileges)

		// Constraints
		pg.Get("/constraints", pgmetaHandler.ListConstraints)
		pg.Get("/primary-keys", pgmetaHandler.ListPrimaryKeys)
		pg.Get("/foreign-keys", pgmetaHandler.ListForeignKeysAll)
		pg.Get("/relationships", pgmetaHandler.ListRelationships)

		// SQL Utilities
		pg.Post("/format", pgmetaHandler.FormatSQL)
		pg.Post("/explain", pgmetaHandler.ExplainQuery)

		// Generators
		pg.Get("/generators/typescript", pgmetaHandler.GenerateTypescript)
		pg.Get("/generators/openapi", pgmetaHandler.GenerateOpenAPI)
		pg.Get("/generators/go", pgmetaHandler.GenerateGo)
		pg.Get("/generators/swift", pgmetaHandler.GenerateSwift)
		pg.Get("/generators/python", pgmetaHandler.GeneratePython)

		// Functions (database functions)
		pg.Get("/functions", pgmetaHandler.ListDatabaseFunctions)
	})

	// Static files for dashboard
	if devMode {
		// In dev mode, proxy to Vite dev server
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			// For SPA routing, return index.html for non-API routes
			return c.Text(200, "Dashboard running on http://localhost:5173")
		})
	} else {
		// In production, serve embedded static files
		staticContent, err := fs.Sub(assets.StaticFS, "static")
		if err != nil {
			return nil, err
		}

		fileServer := http.FileServer(http.FS(staticContent))
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			// Try to serve static file
			path := c.Request().URL.Path
			if path == "/" {
				path = "/index.html"
			}

			// Check if file exists
			if _, err := fs.Stat(staticContent, path[1:]); err == nil {
				fileServer.ServeHTTP(c.Writer(), c.Request())
				return nil
			}

			// SPA fallback - serve index.html
			c.Request().URL.Path = "/index.html"
			fileServer.ServeHTTP(c.Writer(), c.Request())
			return nil
		})
	}

	return app, nil
}

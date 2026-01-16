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

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "healthy"})
	})

	// API key middleware for Supabase compatibility (defined early for reuse)
	apiKeyMw := middleware.APIKey(middleware.DefaultAPIKeyConfig())
	serviceRoleMw := middleware.RequireServiceRole()

	// Auth API (GoTrue compatible)
	app.Group("/auth/v1", func(auth *mizu.Router) {
		// Apply API key middleware to all auth endpoints
		auth.Use(apiKeyMw)

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

	// Database API (Dashboard)
	app.Group("/api/database", func(database *mizu.Router) {
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

	// Functions API
	app.Group("/api/functions", func(functions *mizu.Router) {
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

	// Public function invocation
	app.Post("/functions/v1/{name}", functionsHandler.InvokeFunction)

	// Realtime API
	app.Group("/api/realtime", func(realtime *mizu.Router) {
		realtime.Get("/channels", realtimeHandler.ListChannels)
		realtime.Get("/stats", realtimeHandler.GetStats)
	})

	// WebSocket endpoint for realtime
	app.Get("/realtime/v1/websocket", realtimeHandler.WebSocket)

	// Dashboard API
	app.Group("/api", func(dashboard *mizu.Router) {
		dashboard.Get("/dashboard/stats", dashboardHandler.GetStats)
		dashboard.Get("/dashboard/health", dashboardHandler.GetHealth)
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

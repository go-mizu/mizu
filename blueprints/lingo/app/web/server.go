package web

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/lingo/assets"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
)

// NewServer creates a new HTTP server
func NewServer(st store.Store, devMode bool) (http.Handler, error) {
	app := mizu.New()

	// API handlers
	authHandler := api.NewAuthHandler(st)
	userHandler := api.NewUserHandler(st)
	courseHandler := api.NewCourseHandler(st)
	lessonHandler := api.NewLessonHandler(st)
	progressHandler := api.NewProgressHandler(st)
	leagueHandler := api.NewLeagueHandler(st)
	socialHandler := api.NewSocialHandler(st)
	achievementHandler := api.NewAchievementHandler(st)
	shopHandler := api.NewShopHandler(st)

	// API routes
	app.Group("/api/v1", func(apiGroup *mizu.Router) {
		// Auth endpoints
		apiGroup.Post("/auth/signup", authHandler.Signup)
		apiGroup.Post("/auth/login", authHandler.Login)
		apiGroup.Post("/auth/logout", authHandler.Logout)
		apiGroup.Post("/auth/refresh", authHandler.Refresh)

		// User endpoints
		apiGroup.Get("/users/me", userHandler.GetMe)
		apiGroup.Put("/users/me", userHandler.UpdateMe)
		apiGroup.Get("/users/:username", userHandler.GetByUsername)
		apiGroup.Get("/users/:id/stats", userHandler.GetStats)
		apiGroup.Put("/users/me/settings", userHandler.UpdateSettings)

		// Course endpoints
		apiGroup.Get("/languages", courseHandler.ListLanguages)
		apiGroup.Get("/courses", courseHandler.ListCourses)
		apiGroup.Get("/courses/:id", courseHandler.GetCourse)
		apiGroup.Post("/courses/:id/enroll", courseHandler.Enroll)
		apiGroup.Get("/courses/:id/path", courseHandler.GetPath)
		apiGroup.Get("/units/:id", courseHandler.GetUnit)
		apiGroup.Get("/skills/:id", courseHandler.GetSkill)

		// Lesson endpoints
		apiGroup.Get("/lessons/:id", lessonHandler.GetLesson)
		apiGroup.Post("/lessons/:id/start", lessonHandler.StartLesson)
		apiGroup.Post("/lessons/:id/complete", lessonHandler.CompleteLesson)
		apiGroup.Post("/exercises/:id/answer", lessonHandler.AnswerExercise)

		// Progress endpoints
		apiGroup.Get("/progress", progressHandler.GetProgress)
		apiGroup.Get("/xp/history", progressHandler.GetXPHistory)
		apiGroup.Get("/streaks", progressHandler.GetStreaks)
		apiGroup.Post("/streaks/freeze", progressHandler.UseStreakFreeze)
		apiGroup.Get("/hearts", progressHandler.GetHearts)
		apiGroup.Post("/hearts/refill", progressHandler.RefillHearts)
		apiGroup.Get("/practice/mistakes", progressHandler.GetMistakes)

		// League endpoints
		apiGroup.Get("/leagues", leagueHandler.GetLeagues)
		apiGroup.Get("/leagues/current", leagueHandler.GetCurrentLeague)
		apiGroup.Get("/leagues/leaderboard", leagueHandler.GetLeaderboard)

		// Social endpoints
		apiGroup.Get("/friends", socialHandler.GetFriends)
		apiGroup.Post("/friends/:id/follow", socialHandler.Follow)
		apiGroup.Delete("/friends/:id/unfollow", socialHandler.Unfollow)
		apiGroup.Get("/friends/leaderboard", socialHandler.GetFriendLeaderboard)
		apiGroup.Get("/friends/quests", socialHandler.GetFriendQuests)
		apiGroup.Get("/friends/streaks", socialHandler.GetFriendStreaks)
		apiGroup.Get("/notifications", socialHandler.GetNotifications)
		apiGroup.Put("/notifications/:id/read", socialHandler.MarkNotificationRead)

		// Achievement endpoints
		apiGroup.Get("/achievements", achievementHandler.GetAchievements)
		apiGroup.Get("/achievements/me", achievementHandler.GetMyAchievements)

		// Stories endpoints
		apiGroup.Get("/stories", courseHandler.GetStories)
		apiGroup.Get("/stories/:id", courseHandler.GetStory)
		apiGroup.Post("/stories/:id/complete", courseHandler.CompleteStory)

		// Shop endpoints
		apiGroup.Get("/shop/items", shopHandler.GetItems)
		apiGroup.Post("/shop/purchase", shopHandler.Purchase)
	})

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Serve frontend
	if devMode {
		// Proxy to Vite dev server
		viteURL, _ := url.Parse("http://localhost:5173")
		proxy := httputil.NewSingleHostReverseProxy(viteURL)
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			proxy.ServeHTTP(c.Writer(), c.Request())
			return nil
		})
	} else {
		// Serve embedded static files
		staticContent, err := fs.Sub(assets.StaticFS, "static")
		if err != nil {
			return nil, err
		}

		// Read index.html content for SPA fallback
		indexHTML, err := fs.ReadFile(staticContent, "index.html")
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

			// Check if file exists (must be a file, not directory)
			if info, err := fs.Stat(staticContent, path[1:]); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer(), c.Request())
				return nil
			}

			// SPA fallback - serve index.html content directly
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(200, string(indexHTML))
		})
	}

	return app, nil
}

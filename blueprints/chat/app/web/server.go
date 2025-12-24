// Package web provides the HTTP server for the chat application.
package web

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"
	"github.com/gorilla/websocket"

	"github.com/go-mizu/blueprints/chat/app/web/handler"
	"github.com/go-mizu/blueprints/chat/app/web/ws"
	"github.com/go-mizu/blueprints/chat/assets"
	"github.com/go-mizu/blueprints/chat/feature/accounts"
	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/members"
	"github.com/go-mizu/blueprints/chat/feature/messages"
	"github.com/go-mizu/blueprints/chat/feature/presence"
	"github.com/go-mizu/blueprints/chat/feature/roles"
	"github.com/go-mizu/blueprints/chat/feature/servers"
	"github.com/go-mizu/blueprints/chat/store/duckdb"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app       *mizu.App
	cfg       Config
	db        *sql.DB
	templates map[string]*template.Template
	hub       *ws.Hub
	upgrader  websocket.Upgrader

	// Services
	accounts accounts.API
	servers  servers.API
	channels channels.API
	messages messages.API
	members  members.API
	roles    roles.API
	presence presence.API

	// Handlers
	authHandler    *handler.Auth
	serverHandler  *handler.Server
	channelHandler *handler.Channel
	messageHandler *handler.Message
	pageHandler    *handler.Page
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "chat.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	coreStore, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	if err := coreStore.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create stores
	usersStore := duckdb.NewUsersStore(db)
	serversStore := duckdb.NewServersStore(db)
	channelsStore := duckdb.NewChannelsStore(db)
	messagesStore := duckdb.NewMessagesStore(db)
	membersStore := duckdb.NewMembersStore(db)
	rolesStore := duckdb.NewRolesStore(db)
	presenceStore := duckdb.NewPresenceStore(db)

	// Create services
	accountsSvc := accounts.NewService(usersStore)
	serversSvc := servers.NewService(serversStore)
	channelsSvc := channels.NewService(channelsStore)
	messagesSvc := messages.NewService(messagesStore)
	membersSvc := members.NewService(membersStore)
	rolesSvc := roles.NewService(rolesStore, &memberRoleGetter{membersStore})
	presenceSvc := presence.NewService(presenceStore)

	// Create WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	s := &Server{
		app:       mizu.New(),
		cfg:       cfg,
		db:        db,
		templates: tmpl,
		hub:       hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		accounts: accountsSvc,
		servers:  serversSvc,
		channels: channelsSvc,
		messages: messagesSvc,
		members:  membersSvc,
		roles:    rolesSvc,
		presence: presenceSvc,
	}

	// Create handlers
	s.authHandler = handler.NewAuth(accountsSvc)
	s.serverHandler = handler.NewServer(serversSvc, channelsSvc, membersSvc, rolesSvc, s.getUserID)
	s.channelHandler = handler.NewChannel(channelsSvc, membersSvc, hub, s.getUserID)
	s.messageHandler = handler.NewMessage(messagesSvc, channelsSvc, accountsSvc, presenceSvc, hub, s.getUserID)
	s.pageHandler = handler.NewPage(tmpl, accountsSvc, serversSvc, channelsSvc, messagesSvc, membersSvc, s.getUserID, cfg.Dev)

	s.setupRoutes()

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	log.Printf("Starting server on %s", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	s.hub.Stop()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() *mizu.App {
	return s.app
}

func (s *Server) setupRoutes() {
	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandler.Register)
		api.Post("/auth/login", s.authHandler.Login)
		api.Post("/auth/logout", s.authRequired(func(c *mizu.Ctx) error {
			return s.authHandler.Logout(c, s.getUserID(c))
		}))
		api.Get("/auth/me", s.authRequired(func(c *mizu.Ctx) error {
			return s.authHandler.Me(c, s.getUserID(c))
		}))
		api.Patch("/auth/me", s.authRequired(func(c *mizu.Ctx) error {
			return s.authHandler.UpdateMe(c, s.getUserID(c))
		}))

		// Users
		api.Get("/users/search", s.userSearch)

		// Servers
		api.Get("/servers", s.serverHandler.List)
		api.Post("/servers", s.serverHandler.Create)
		api.Get("/servers/public", s.serverHandler.ListPublic)
		api.Get("/servers/{id}", s.serverHandler.Get)
		api.Patch("/servers/{id}", s.serverHandler.Update)
		api.Delete("/servers/{id}", s.serverHandler.Delete)
		api.Get("/servers/{id}/channels", s.serverHandler.ListChannels)
		api.Post("/servers/{id}/channels", s.channelHandler.Create)
		api.Get("/servers/{id}/categories", s.channelHandler.ListCategories)
		api.Post("/servers/{id}/categories", s.channelHandler.CreateCategory)
		api.Get("/servers/{id}/members", s.serverHandler.ListMembers)
		api.Post("/servers/{id}/join", s.serverHandler.Join)
		api.Delete("/servers/{id}/leave", s.serverHandler.Leave)
		api.Get("/servers/{id}/roles", s.serverHandler.ListRoles)

		// Invites
		api.Post("/invites/{code}", s.serverHandler.JoinByInvite)

		// Channels
		api.Get("/channels/{id}", s.channelHandler.Get)
		api.Patch("/channels/{id}", s.channelHandler.Update)
		api.Delete("/channels/{id}", s.channelHandler.Delete)
		api.Get("/channels/{id}/messages", s.messageHandler.List)
		api.Post("/channels/{id}/messages", s.messageHandler.Create)
		api.Get("/channels/{id}/messages/{msg_id}", s.messageHandler.Get)
		api.Patch("/channels/{id}/messages/{msg_id}", s.messageHandler.Update)
		api.Delete("/channels/{id}/messages/{msg_id}", s.messageHandler.Delete)
		api.Get("/channels/{id}/pins", s.messageHandler.ListPinned)
		api.Put("/channels/{id}/pins/{msg_id}", s.messageHandler.Pin)
		api.Delete("/channels/{id}/pins/{msg_id}", s.messageHandler.Unpin)
		api.Post("/channels/{id}/typing", s.messageHandler.Typing)
		api.Put("/channels/{id}/messages/{msg_id}/reactions/{emoji}", s.messageHandler.AddReaction)
		api.Delete("/channels/{id}/messages/{msg_id}/reactions/{emoji}", s.messageHandler.RemoveReaction)

		// DMs
		api.Get("/users/@me/channels", s.channelHandler.ListDMs)
		api.Post("/users/@me/channels", s.channelHandler.CreateDM)

		// Search
		api.Get("/search/messages", s.messageHandler.Search)
	})

	// WebSocket
	s.app.Get("/ws", s.handleWebSocket)

	// Web routes
	s.app.Get("/", s.pageHandler.Home)
	s.app.Get("/login", s.pageHandler.Login)
	s.app.Get("/register", s.pageHandler.Register)
	s.app.Get("/explore", s.pageHandler.Explore)
	s.app.Get("/settings", s.pageHandler.Settings)
	s.app.Get("/channels/{server_id}/{channel_id}", s.pageHandler.ServerView)

	// Static files
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})
}

func (s *Server) handleWebSocket(c *mizu.Ctx) error {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		return handler.Unauthorized(c, "Token required")
	}

	// Validate session
	session, err := s.accounts.GetSession(c.Request().Context(), token)
	if err != nil {
		return handler.Unauthorized(c, "Invalid session")
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(c.Writer(), c.Request(), nil)
	if err != nil {
		return err
	}

	// Create WebSocket connection
	wsConn := ws.NewConnection(s.hub, conn, session.UserID, session.ID)

	// Register and start
	s.hub.Register(wsConn)
	wsConn.Start()

	// Subscribe to user's servers
	ctx := c.Request().Context()
	srvs, _ := s.servers.ListByUser(ctx, session.UserID, 100, 0)
	for _, srv := range srvs {
		s.hub.SubscribeToServer(wsConn, srv.ID)
	}

	// Subscribe to user's DMs
	dms, _ := s.channels.ListDMsByUser(ctx, session.UserID)
	for _, dm := range dms {
		s.hub.SubscribeToChannel(wsConn, dm.ID)
	}

	// Set user online
	s.presence.SetOnline(ctx, session.UserID)

	// Send READY event
	user, _ := s.accounts.GetByID(ctx, session.UserID)
	wsConn.SendEvent(ws.EventReady, map[string]any{
		"user":     user,
		"servers":  srvs,
		"channels": dms,
	})

	return nil
}

func (s *Server) userSearch(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return handler.BadRequest(c, "Query required")
	}

	users, err := s.accounts.Search(c.Request().Context(), query, 25)
	if err != nil {
		return handler.InternalError(c, "Search failed")
	}

	return handler.Success(c, users)
}

// getUserID extracts the user ID from the request.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// Try Authorization header first
	auth := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.accounts.GetSession(c.Request().Context(), token)
		if err == nil {
			return session.UserID
		}
	}

	// Try cookie
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil {
			return session.UserID
		}
	}

	return ""
}

// authRequired requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		userID := s.getUserID(c)
		if userID == "" {
			return handler.Unauthorized(c, "Authentication required")
		}
		return next(c)
	}
}

// memberRoleGetter implements roles.MemberRoleGetter.
type memberRoleGetter struct {
	store *duckdb.MembersStore
}

func (g *memberRoleGetter) GetMemberRoleIDs(ctx context.Context, serverID, userID string) ([]string, error) {
	member, err := g.store.Get(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}
	return member.RoleIDs, nil
}

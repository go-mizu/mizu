// Package web provides the HTTP server for the messaging application.
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

	"github.com/go-mizu/blueprints/messaging/app/web/handler"
	"github.com/go-mizu/blueprints/messaging/app/web/ws"
	"github.com/go-mizu/blueprints/messaging/assets"
	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/contacts"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
	"github.com/go-mizu/blueprints/messaging/feature/presence"
	"github.com/go-mizu/blueprints/messaging/feature/stories"
	"github.com/go-mizu/blueprints/messaging/store/duckdb"
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
	contacts contacts.API
	chats    chats.API
	messages messages.API
	stories  stories.API
	presence presence.API

	// Handlers
	authHandler    *handler.Auth
	chatHandler    *handler.Chat
	messageHandler *handler.Message
	storyHandler   *handler.Story
	pageHandler    *handler.Page
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "messaging.duckdb")
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
	chatsStore := duckdb.NewChatsStore(db)
	messagesStore := duckdb.NewMessagesStore(db)

	// Create services
	accountsSvc := accounts.NewService(usersStore)
	contactsSvc := contacts.NewService(nil) // TODO: implement contacts store
	chatsSvc := chats.NewService(chatsStore)
	messagesSvc := messages.NewService(messagesStore)
	storiesSvc := stories.NewService(nil) // TODO: implement stories store
	presenceSvc := presence.NewService(nil) // TODO: implement presence store

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
		contacts: contactsSvc,
		chats:    chatsSvc,
		messages: messagesSvc,
		stories:  storiesSvc,
		presence: presenceSvc,
	}

	// Create handlers
	s.authHandler = handler.NewAuth(accountsSvc)
	s.chatHandler = handler.NewChat(chatsSvc, s.getUserID)
	s.messageHandler = handler.NewMessage(messagesSvc, chatsSvc, accountsSvc, hub, s.getUserID)
	s.storyHandler = handler.NewStory(storiesSvc, s.getUserID)
	s.pageHandler = handler.NewPage(tmpl, accountsSvc, chatsSvc, messagesSvc, s.getUserID, cfg.Dev)

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
		api.Get("/users/{id}", s.getUser)

		// Contacts
		api.Get("/contacts", s.authRequired(s.listContacts))
		api.Post("/contacts", s.authRequired(s.addContact))
		api.Delete("/contacts/{id}", s.authRequired(s.removeContact))
		api.Post("/contacts/{id}/block", s.authRequired(s.blockContact))
		api.Delete("/contacts/{id}/block", s.authRequired(s.unblockContact))

		// Chats
		api.Get("/chats", s.chatHandler.List)
		api.Post("/chats", s.chatHandler.Create)
		api.Get("/chats/{id}", s.chatHandler.Get)
		api.Patch("/chats/{id}", s.chatHandler.Update)
		api.Delete("/chats/{id}", s.chatHandler.Delete)
		api.Post("/chats/{id}/archive", s.chatHandler.Archive)
		api.Delete("/chats/{id}/archive", s.chatHandler.Unarchive)
		api.Post("/chats/{id}/pin", s.chatHandler.Pin)
		api.Delete("/chats/{id}/pin", s.chatHandler.Unpin)
		api.Post("/chats/{id}/mute", s.chatHandler.Mute)
		api.Delete("/chats/{id}/mute", s.chatHandler.Unmute)
		api.Post("/chats/{id}/read", s.chatHandler.MarkAsRead)
		api.Post("/chats/{id}/typing", s.messageHandler.Typing)

		// Messages
		api.Get("/chats/{id}/messages", s.messageHandler.List)
		api.Post("/chats/{id}/messages", s.messageHandler.Create)
		api.Get("/chats/{id}/messages/{msg_id}", s.messageHandler.Get)
		api.Patch("/chats/{id}/messages/{msg_id}", s.messageHandler.Update)
		api.Delete("/chats/{id}/messages/{msg_id}", s.messageHandler.Delete)
		api.Post("/chats/{id}/messages/{msg_id}/react", s.messageHandler.AddReaction)
		api.Delete("/chats/{id}/messages/{msg_id}/react", s.messageHandler.RemoveReaction)
		api.Post("/chats/{id}/messages/{msg_id}/forward", s.messageHandler.Forward)
		api.Post("/chats/{id}/messages/{msg_id}/star", s.messageHandler.Star)
		api.Delete("/chats/{id}/messages/{msg_id}/star", s.messageHandler.Unstar)
		api.Get("/chats/{id}/pins", s.messageHandler.ListPinned)
		api.Post("/chats/{id}/pins/{msg_id}", s.messageHandler.Pin)
		api.Delete("/chats/{id}/pins/{msg_id}", s.messageHandler.Unpin)

		// Stories
		api.Get("/stories", s.storyHandler.List)
		api.Post("/stories", s.storyHandler.Create)
		api.Get("/stories/{id}", s.storyHandler.Get)
		api.Delete("/stories/{id}", s.storyHandler.Delete)
		api.Post("/stories/{id}/view", s.storyHandler.View)
		api.Get("/stories/{id}/viewers", s.storyHandler.GetViewers)

		// Search
		api.Get("/search/messages", s.messageHandler.Search)
		api.Get("/starred", s.messageHandler.ListStarred)
	})

	// WebSocket
	s.app.Get("/ws", s.handleWebSocket)

	// Web routes
	s.app.Get("/", s.pageHandler.Home)
	s.app.Get("/login", s.pageHandler.Login)
	s.app.Get("/register", s.pageHandler.Register)
	s.app.Get("/app", s.pageHandler.App)
	s.app.Get("/chat/{id}", s.pageHandler.ChatView)
	s.app.Get("/settings", s.pageHandler.Settings)

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
	token := c.Query("token")
	if token == "" {
		return handler.Unauthorized(c, "Token required")
	}

	session, err := s.accounts.GetSession(c.Request().Context(), token)
	if err != nil {
		return handler.Unauthorized(c, "Invalid session")
	}

	conn, err := s.upgrader.Upgrade(c.Writer(), c.Request(), nil)
	if err != nil {
		return err
	}

	wsConn := ws.NewConnection(s.hub, conn, session.UserID, session.ID)
	s.hub.Register(wsConn)
	wsConn.Start()

	// Subscribe to user's chats
	ctx := c.Request().Context()
	userChats, _ := s.chats.List(ctx, session.UserID, chats.ListOpts{Limit: 100})
	for _, chat := range userChats {
		s.hub.SubscribeToChat(wsConn, chat.ID)
	}

	// Set user online
	s.presence.SetOnline(ctx, session.UserID)

	// Send READY event
	user, _ := s.accounts.GetByID(ctx, session.UserID)
	wsConn.SendEvent(ws.EventReady, map[string]any{
		"user":  user,
		"chats": userChats,
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

func (s *Server) getUser(c *mizu.Ctx) error {
	userID := c.Param("id")
	user, err := s.accounts.GetByID(c.Request().Context(), userID)
	if err != nil {
		return handler.NotFound(c, "User not found")
	}
	return handler.Success(c, user)
}

func (s *Server) listContacts(c *mizu.Ctx) error {
	userID := s.getUserID(c)
	contactsList, err := s.contacts.List(c.Request().Context(), userID)
	if err != nil {
		return handler.InternalError(c, "Failed to list contacts")
	}
	return handler.Success(c, contactsList)
}

func (s *Server) addContact(c *mizu.Ctx) error {
	userID := s.getUserID(c)
	var in contacts.AddIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return handler.BadRequest(c, "Invalid request")
	}
	contact, err := s.contacts.Add(c.Request().Context(), userID, &in)
	if err != nil {
		return handler.BadRequest(c, err.Error())
	}
	return handler.Success(c, contact)
}

func (s *Server) removeContact(c *mizu.Ctx) error {
	userID := s.getUserID(c)
	contactID := c.Param("id")
	if err := s.contacts.Remove(c.Request().Context(), userID, contactID); err != nil {
		return handler.InternalError(c, "Failed to remove contact")
	}
	return handler.Success(c, nil)
}

func (s *Server) blockContact(c *mizu.Ctx) error {
	userID := s.getUserID(c)
	contactID := c.Param("id")
	if err := s.contacts.Block(c.Request().Context(), userID, contactID); err != nil {
		return handler.InternalError(c, "Failed to block contact")
	}
	return handler.Success(c, nil)
}

func (s *Server) unblockContact(c *mizu.Ctx) error {
	userID := s.getUserID(c)
	contactID := c.Param("id")
	if err := s.contacts.Unblock(c.Request().Context(), userID, contactID); err != nil {
		return handler.InternalError(c, "Failed to unblock contact")
	}
	return handler.Success(c, nil)
}

func (s *Server) getUserID(c *mizu.Ctx) string {
	auth := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.accounts.GetSession(c.Request().Context(), token)
		if err == nil {
			return session.UserID
		}
	}

	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil {
			return session.UserID
		}
	}

	return ""
}

func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		userID := s.getUserID(c)
		if userID == "" {
			return handler.Unauthorized(c, "Authentication required")
		}
		return next(c)
	}
}

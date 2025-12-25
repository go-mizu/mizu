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
	"github.com/go-mizu/blueprints/messaging/feature/friendcode"
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

// AgentUsername is the username for the system agent user.
const AgentUsername = "mizu-agent"

// Server is the HTTP server.
type Server struct {
	app       *mizu.App
	cfg       Config
	db        *sql.DB
	templates map[string]*template.Template
	hub       *ws.Hub
	upgrader  websocket.Upgrader

	// Services
	accounts   accounts.API
	contacts   contacts.API
	chats      chats.API
	messages   messages.API
	stories    stories.API
	presence   presence.API
	friendcode friendcode.API

	// System users
	agentID string

	// Handlers
	authHandler       *handler.Auth
	chatHandler       *handler.Chat
	messageHandler    *handler.Message
	storyHandler      *handler.Story
	pageHandler       *handler.Page
	friendcodeHandler *handler.FriendCode
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
	friendCodesStore := duckdb.NewFriendCodesStore(db)
	friendCodeUserStore := duckdb.NewFriendCodeUserStore(usersStore)
	friendCodeContactStore := duckdb.NewFriendCodeContactStore(db)

	// Create services
	accountsSvc := accounts.NewService(usersStore)
	contactsSvc := contacts.NewService(nil) // TODO: implement contacts store
	chatsSvc := chats.NewService(chatsStore)
	messagesSvc := messages.NewService(messagesStore)
	storiesSvc := stories.NewService(nil) // TODO: implement stories store
	presenceSvc := presence.NewService(nil) // TODO: implement presence store
	friendcodeSvc := friendcode.NewService(friendCodesStore, friendCodeUserStore, friendCodeContactStore, "")

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
		accounts:   accountsSvc,
		contacts:   contactsSvc,
		chats:      chatsSvc,
		messages:   messagesSvc,
		stories:    storiesSvc,
		presence:   presenceSvc,
		friendcode: friendcodeSvc,
	}

	// Ensure the system agent user exists
	if err := s.ensureAgent(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure agent: %v", err)
	}

	// Create handlers
	s.authHandler = handler.NewAuth(accountsSvc, s.setupNewUser)
	s.chatHandler = handler.NewChat(chatsSvc, accountsSvc, messagesSvc, s.getUserID)
	s.messageHandler = handler.NewMessage(messagesSvc, chatsSvc, accountsSvc, hub, s.getUserID)
	s.storyHandler = handler.NewStory(storiesSvc, s.getUserID)
	s.pageHandler = handler.NewPage(tmpl, accountsSvc, chatsSvc, messagesSvc, s.getUserID, cfg.Dev)
	s.friendcodeHandler = handler.NewFriendCode(friendcodeSvc, s.getUserID)

	s.setupRoutes()

	return s, nil
}

// ensureAgent creates or retrieves the Mizu Agent system user.
func (s *Server) ensureAgent(ctx context.Context) error {
	// Try to get existing agent
	agent, err := s.accounts.GetByUsername(ctx, AgentUsername)
	if err == nil && agent != nil {
		s.agentID = agent.ID
		return nil
	}

	// Create the agent user
	agent, err = s.accounts.Create(ctx, &accounts.CreateIn{
		Username:    AgentUsername,
		Email:       "agent@mizu.dev",
		Password:    "agent-system-password-not-for-login",
		DisplayName: "Mizu Agent",
	})
	if err != nil && err != accounts.ErrUsernameTaken {
		return err
	}
	if err == accounts.ErrUsernameTaken {
		agent, err = s.accounts.GetByUsername(ctx, AgentUsername)
		if err != nil {
			return err
		}
	}

	s.agentID = agent.ID
	return nil
}

// setupNewUser creates the default chats for a new user:
// 1. Saved Messages (self-chat) with a welcome message
// 2. Chat with Mizu Agent with a welcome message
func (s *Server) setupNewUser(ctx context.Context, userID string) {
	// Create Saved Messages (self-chat)
	savedChat, err := s.chats.CreateDirect(ctx, userID, &chats.CreateDirectIn{
		RecipientID: userID,
	})
	if err == nil && savedChat != nil {
		// Add a welcome message to Saved Messages
		s.messages.Create(ctx, userID, &messages.CreateIn{
			ChatID:  savedChat.ID,
			Type:    messages.TypeText,
			Content: "Welcome to Saved Messages! Use this space to save notes, links, and reminders to yourself.",
		})
	}

	// Create chat with Mizu Agent (if agent exists)
	if s.agentID != "" {
		agentChat, err := s.chats.CreateDirect(ctx, userID, &chats.CreateDirectIn{
			RecipientID: s.agentID,
		})
		if err == nil && agentChat != nil {
			// Add a welcome message from the agent
			s.messages.Create(ctx, s.agentID, &messages.CreateIn{
				ChatID:  agentChat.ID,
				Type:    messages.TypeText,
				Content: "Hello! I'm Mizu Agent, your friendly assistant. I'm here to help you get started with messaging. Feel free to ask me anything!",
			})
		}
	}
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
		api.Get("/users/me", s.authRequired(func(c *mizu.Ctx) error {
			return s.authHandler.Me(c, s.getUserID(c))
		}))
		api.Post("/users/ensure-chats", s.authRequired(s.ensureUserChats))
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

		// Friend Codes (QR Code Friend Feature)
		api.Get("/friend-code", s.friendcodeHandler.Generate)
		api.Delete("/friend-code", s.friendcodeHandler.Revoke)
		api.Get("/friend-code/{code}", s.friendcodeHandler.Resolve)
		api.Post("/friend-code/{code}", s.friendcodeHandler.AddFriend)
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
	s.app.Get("/add-friend/{code}", s.handleAddFriend)

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
	// Try query param first, then cookie
	token := c.Query("token")
	if token == "" {
		cookie, err := c.Cookie("session")
		if err == nil && cookie.Value != "" {
			token = cookie.Value
		}
	}
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

// ensureUserChats ensures the user has their default chats (Saved Messages and Agent chat).
func (s *Server) ensureUserChats(c *mizu.Ctx) error {
	userID := s.getUserID(c)
	ctx := c.Request().Context()

	created := []string{}

	// Check for Saved Messages (self-chat)
	_, err := s.chats.GetDirectChat(ctx, userID, userID)
	if err == chats.ErrNotFound {
		savedChat, err := s.chats.CreateDirect(ctx, userID, &chats.CreateDirectIn{
			RecipientID: userID,
		})
		if err == nil && savedChat != nil {
			s.messages.Create(ctx, userID, &messages.CreateIn{
				ChatID:  savedChat.ID,
				Type:    messages.TypeText,
				Content: "Welcome to Saved Messages! Use this space to save notes, links, and reminders to yourself.",
			})
			created = append(created, "saved_messages")
		}
	}

	// Check for Agent chat
	if s.agentID != "" && s.agentID != userID {
		_, err := s.chats.GetDirectChat(ctx, userID, s.agentID)
		if err == chats.ErrNotFound {
			agentChat, err := s.chats.CreateDirect(ctx, userID, &chats.CreateDirectIn{
				RecipientID: s.agentID,
			})
			if err == nil && agentChat != nil {
				s.messages.Create(ctx, s.agentID, &messages.CreateIn{
					ChatID:  agentChat.ID,
					Type:    messages.TypeText,
					Content: "Hello! I'm Mizu Agent, your friendly assistant. I'm here to help you get started with messaging. Feel free to ask me anything!",
				})
				created = append(created, "agent_chat")
			}
		}
	}

	return handler.Success(c, map[string]any{
		"created": created,
	})
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

// handleAddFriend handles the /add-friend/{code} deep link.
// It redirects logged-in users to /app with the friend code,
// or redirects to login for unauthenticated users.
func (s *Server) handleAddFriend(c *mizu.Ctx) error {
	code := c.Param("code")
	userID := s.getUserID(c)

	if userID == "" {
		// Redirect to login, preserving the friend code
		return c.Redirect(http.StatusFound, "/login?next=/add-friend/"+code)
	}

	// Redirect to app with friend code to trigger the add friend modal
	return c.Redirect(http.StatusFound, "/app?add-friend="+code)
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

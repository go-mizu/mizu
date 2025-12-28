package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

// contextKey is a type for context keys
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey contextKey = "user"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	users users.API
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(users users.API) *AuthHandler {
	return &AuthHandler{users: users}
}

// GetUser returns the authenticated user from context
func GetUser(ctx context.Context) *users.User {
	u, _ := ctx.Value(UserContextKey).(*users.User)
	return u
}

// GetUserID returns the authenticated user ID from context
func GetUserID(ctx context.Context) int64 {
	u := GetUser(ctx)
	if u == nil {
		return 0
	}
	return u.ID
}

// RequireAuth is middleware that requires authentication
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := h.extractUser(r)
		if user == nil {
			WriteUnauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is middleware that optionally authenticates
func (h *AuthHandler) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := h.extractUser(r)
		if user != nil {
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// extractUser extracts user from Authorization header
func (h *AuthHandler) extractUser(r *http.Request) *users.User {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil
	}

	// Support Basic auth and Bearer token
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return nil
	}

	switch strings.ToLower(parts[0]) {
	case "basic":
		return h.extractBasicAuth(r, parts[1])
	case "bearer", "token":
		return h.extractTokenAuth(r, parts[1])
	default:
		return nil
	}
}

// extractBasicAuth extracts user from Basic auth
func (h *AuthHandler) extractBasicAuth(r *http.Request, encoded string) *users.User {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil
	}

	user, err := h.users.Authenticate(r.Context(), username, password)
	if err != nil {
		return nil
	}
	return user
}

// extractTokenAuth extracts user from Bearer token
func (h *AuthHandler) extractTokenAuth(r *http.Request, token string) *users.User {
	// TODO: Implement token-based authentication
	// For now, treat token as user login for simplicity
	user, err := h.users.GetByLogin(r.Context(), token)
	if err != nil {
		return nil
	}
	return user
}

// Login handles POST /login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	user, err := h.users.Authenticate(r.Context(), in.Login, in.Password)
	if err != nil {
		WriteUnauthorized(w)
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

// Register handles POST /register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in users.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	user, err := h.users.Create(r.Context(), &in)
	if err != nil {
		switch err {
		case users.ErrUserExists:
			WriteConflict(w, "Login already exists")
		case users.ErrEmailExists:
			WriteConflict(w, "Email already exists")
		default:
			WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	WriteCreated(w, user)
}

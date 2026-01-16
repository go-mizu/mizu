package flight

import (
	"context"
	"strings"

	"github.com/apache/arrow-go/v18/arrow/flight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthConfig configures authentication for the Flight server.
type AuthConfig struct {
	// TokenValidator validates bearer tokens and returns claims.
	// The claims are stored in the context and can be retrieved with Claims().
	TokenValidator func(token string) (map[string]any, error)

	// BasicAuthValidator validates username/password and returns a bearer token.
	// Used during the Handshake RPC.
	BasicAuthValidator func(username, password string) (string, error)

	// AllowUnauthenticated permits requests without tokens.
	AllowUnauthenticated bool
}

// claimsKey is the context key for authentication claims.
type claimsKey struct{}

// Claims returns the authentication claims from the context.
func Claims(ctx context.Context) map[string]any {
	claims, _ := ctx.Value(claimsKey{}).(map[string]any)
	return claims
}

// ContextWithClaims returns a new context with the given claims.
func ContextWithClaims(ctx context.Context, claims map[string]any) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

// serverAuthHandler implements flight.ServerAuthHandler.
type serverAuthHandler struct {
	cfg *AuthConfig
}

// NewServerAuthHandler creates a new server auth handler.
func NewServerAuthHandler(cfg *AuthConfig) flight.ServerAuthHandler {
	return &serverAuthHandler{cfg: cfg}
}

// Authenticate handles the initial authentication handshake.
func (h *serverAuthHandler) Authenticate(conn flight.AuthConn) error {
	if h.cfg == nil || h.cfg.BasicAuthValidator == nil {
		if h.cfg != nil && h.cfg.AllowUnauthenticated {
			return nil
		}
		return status.Error(codes.Unauthenticated, "authentication not configured")
	}

	// Read credentials
	in, err := conn.Read()
	if err != nil {
		return status.Error(codes.Unauthenticated, "failed to read credentials")
	}

	// Parse basic auth: "username:password"
	creds := string(in)
	parts := strings.SplitN(creds, ":", 2)
	if len(parts) != 2 {
		return status.Error(codes.Unauthenticated, "invalid credentials format")
	}

	username, password := parts[0], parts[1]

	// Validate and get token
	token, err := h.cfg.BasicAuthValidator(username, password)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	// Send token back
	return conn.Send([]byte(token))
}

// IsValid validates a token on subsequent requests.
func (h *serverAuthHandler) IsValid(token string) (any, error) {
	if h.cfg == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication not configured")
	}

	if h.cfg.AllowUnauthenticated && token == "" {
		return nil, nil
	}

	if h.cfg.TokenValidator == nil {
		return nil, status.Error(codes.Unauthenticated, "token validation not configured")
	}

	claims, err := h.cfg.TokenValidator(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return claims, nil
}

// authMiddleware provides gRPC middleware for authentication.
type authMiddleware struct {
	cfg *AuthConfig
}

// CreateAuthMiddleware creates Flight server middleware for authentication.
func CreateAuthMiddleware(cfg *AuthConfig) flight.ServerMiddleware {
	m := &authMiddleware{cfg: cfg}
	return flight.CreateServerMiddleware(m)
}

// StartCall is called at the start of each RPC.
func (m *authMiddleware) StartCall(ctx context.Context) context.Context {
	if m.cfg == nil {
		return ctx
	}

	// Extract token from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if m.cfg.AllowUnauthenticated {
			return ctx
		}
		return ctx
	}

	// Check authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		if m.cfg.AllowUnauthenticated {
			return ctx
		}
		return ctx
	}

	token := authHeaders[0]
	// Strip "Bearer " prefix if present
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = token[7:]
	}

	if m.cfg.TokenValidator != nil {
		claims, err := m.cfg.TokenValidator(token)
		if err == nil && claims != nil {
			ctx = ContextWithClaims(ctx, claims)
		}
	}

	return ctx
}

// CallCompleted is called when an RPC completes.
func (m *authMiddleware) CallCompleted(ctx context.Context, err error) {
	// No-op
}

// UnaryAuthInterceptor returns a unary server interceptor for authentication.
func UnaryAuthInterceptor(cfg *AuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, err := authenticateContext(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor returns a stream server interceptor for authentication.
func StreamAuthInterceptor(cfg *AuthConfig) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := authenticateContext(ss.Context(), cfg)
		if err != nil {
			return err
		}
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// authenticateContext validates the token from context and returns claims.
func authenticateContext(ctx context.Context, cfg *AuthConfig) (context.Context, error) {
	if cfg == nil {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if cfg.AllowUnauthenticated {
			return ctx, nil
		}
		return ctx, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		if cfg.AllowUnauthenticated {
			return ctx, nil
		}
		return ctx, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := authHeaders[0]
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = token[7:]
	}

	if cfg.TokenValidator == nil {
		if cfg.AllowUnauthenticated {
			return ctx, nil
		}
		return ctx, status.Error(codes.Unauthenticated, "token validation not configured")
	}

	claims, err := cfg.TokenValidator(token)
	if err != nil {
		return ctx, status.Error(codes.Unauthenticated, err.Error())
	}

	return ContextWithClaims(ctx, claims), nil
}

// ClientAuthHandler implements flight.ClientAuthHandler for client-side authentication.
type ClientAuthHandler struct {
	// Username for basic authentication.
	Username string

	// Password for basic authentication.
	Password string

	// Token is the bearer token to use for requests.
	// If set, basic auth is skipped.
	Token string
}

// Authenticate performs the authentication handshake.
func (h *ClientAuthHandler) Authenticate(ctx context.Context, conn flight.AuthConn) error {
	if h.Token != "" {
		// Already have a token, no need to authenticate
		return nil
	}

	if h.Username == "" {
		return nil
	}

	// Send credentials
	creds := h.Username + ":" + h.Password
	if err := conn.Send([]byte(creds)); err != nil {
		return err
	}

	// Receive token
	token, err := conn.Read()
	if err != nil {
		return err
	}

	h.Token = string(token)
	return nil
}

// GetToken returns the bearer token for requests.
func (h *ClientAuthHandler) GetToken(ctx context.Context) (string, error) {
	return h.Token, nil
}

// TokenCredentials implements grpc.PerRPCCredentials for bearer token auth.
type TokenCredentials struct {
	Token    string
	Insecure bool
}

// GetRequestMetadata returns the authorization header.
func (t *TokenCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.Token,
	}, nil
}

// RequireTransportSecurity returns whether TLS is required.
func (t *TokenCredentials) RequireTransportSecurity() bool {
	return !t.Insecure
}

// File: lib/storage/transport/grpc/auth.go

package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// claimsKey is the context key for authentication claims.
type claimsKey struct{}

// AuthConfig configures authentication for the gRPC server.
type AuthConfig struct {
	// TokenValidator validates the token and returns claims.
	// The token is passed without the "Bearer " prefix.
	TokenValidator func(token string) (map[string]any, error)

	// AllowUnauthenticated permits requests without tokens.
	AllowUnauthenticated bool
}

// Claims extracts authentication claims from the context.
func Claims(ctx context.Context) map[string]any {
	claims, _ := ctx.Value(claimsKey{}).(map[string]any)
	return claims
}

// UnaryAuthInterceptor returns a gRPC unary interceptor for authentication.
func UnaryAuthInterceptor(cfg *AuthConfig) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx, err := authenticate(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor returns a gRPC stream interceptor for authentication.
func StreamAuthInterceptor(cfg *AuthConfig) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx, err := authenticate(ss.Context(), cfg)
		if err != nil {
			return err
		}
		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
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

// authenticate validates the authorization token and returns a context with claims.
func authenticate(ctx context.Context, cfg *AuthConfig) (context.Context, error) {
	if cfg == nil || cfg.TokenValidator == nil {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if cfg.AllowUnauthenticated {
			return ctx, nil
		}
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authValues := md.Get("authorization")
	if len(authValues) == 0 {
		if cfg.AllowUnauthenticated {
			return ctx, nil
		}
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := authValues[0]
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = token[7:]
	}

	claims, err := cfg.TokenValidator(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return context.WithValue(ctx, claimsKey{}, claims), nil
}

// TokenCredentials implements credentials.PerRPCCredentials for client auth.
type TokenCredentials struct {
	Token    string
	Insecure bool
}

// GetRequestMetadata returns the authorization metadata.
func (t *TokenCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.Token,
	}, nil
}

// RequireTransportSecurity indicates whether transport security is required.
func (t *TokenCredentials) RequireTransportSecurity() bool {
	return !t.Insecure
}

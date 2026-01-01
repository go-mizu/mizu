package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
)

// Response is the standard API response.
type Response struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error is an API error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OK returns a 200 response.
func OK(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, Response{Data: data})
}

// Created returns a 201 response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, Response{Data: data})
}

// NoContent returns a 204 response.
func NoContent(c *mizu.Ctx) error {
	return c.NoContent()
}

// BadRequest returns a 400 response.
func BadRequest(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusBadRequest, Response{
		Error: &Error{Code: "BAD_REQUEST", Message: message},
	})
}

// Unauthorized returns a 401 response.
func Unauthorized(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusUnauthorized, Response{
		Error: &Error{Code: "UNAUTHORIZED", Message: message},
	})
}

// Forbidden returns a 403 response.
func Forbidden(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusForbidden, Response{
		Error: &Error{Code: "FORBIDDEN", Message: message},
	})
}

// NotFound returns a 404 response.
func NotFound(c *mizu.Ctx, resource string) error {
	return c.JSON(http.StatusNotFound, Response{
		Error: &Error{Code: "NOT_FOUND", Message: resource + " not found"},
	})
}

// Conflict returns a 409 response.
func Conflict(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusConflict, Response{
		Error: &Error{Code: "CONFLICT", Message: message},
	})
}

// InternalError returns a 500 response.
func InternalError(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusInternalServerError, Response{
		Error: &Error{Code: "INTERNAL_ERROR", Message: message},
	})
}

// Context keys
type contextKey string

const (
	accountKey contextKey = "account"
)

// GetAccount retrieves the account from context.
func GetAccount(c *mizu.Ctx) *accounts.Account {
	if v := c.Request().Context().Value(accountKey); v != nil {
		return v.(*accounts.Account)
	}
	return nil
}

// GetAccountID retrieves the account ID from context.
func GetAccountID(c *mizu.Ctx) string {
	if a := GetAccount(c); a != nil {
		return a.ID
	}
	return ""
}

// RequireAuth is middleware that requires authentication.
func RequireAuth(accountsSvc accounts.API, next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		// Get session token from cookie
		cookie, err := c.Request().Cookie("session")
		if err != nil {
			return Unauthorized(c, "Authentication required")
		}

		account, _, err := accountsSvc.GetByToken(c.Request().Context(), cookie.Value)
		if err != nil {
			return Unauthorized(c, "Invalid or expired session")
		}

		// Store account in context
		ctx := context.WithValue(c.Request().Context(), accountKey, account)
		req := c.Request().WithContext(ctx)
		*c.Request() = *req

		return next(c)
	}
}

// Logger is logging middleware.
func Logger(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		start := time.Now()
		err := next(c)
		duration := time.Since(start)

		log.Printf("%s %s %d %s",
			c.Request().Method,
			c.Request().URL.Path,
			c.StatusCode(),
			duration,
		)

		return err
	}
}

// Recover is panic recovery middleware.
func Recover(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic: %v\n%s", r, debug.Stack())
				InternalError(c, fmt.Sprintf("panic: %v", r))
			}
		}()
		return next(c)
	}
}

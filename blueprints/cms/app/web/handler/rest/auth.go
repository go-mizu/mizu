package rest

import (
	"net/http"
	"strings"

	"github.com/go-mizu/blueprints/cms/feature/auth"
	"github.com/go-mizu/mizu"
)

// Auth handles authentication endpoints.
type Auth struct {
	service auth.API
}

// NewAuth creates a new Auth handler.
func NewAuth(service auth.API) *Auth {
	return &Auth{service: service}
}

// Login handles POST /api/{collection}/login
func (h *Auth) Login(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var input auth.LoginInput
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	result, err := h.service.Login(c.Context(), collection, &input)
	if err != nil {
		return authErrorResponse(c, err)
	}

	// Set cookie if configured
	c.SetCookie(&http.Cookie{
		Name:     "payload-token",
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]any{
		"user":         result.User,
		"token":        result.Token,
		"refreshToken": result.RefreshToken,
		"exp":          result.Exp,
		"message":      "Auth Passed",
	})
}

// Logout handles POST /api/{collection}/logout
func (h *Auth) Logout(c *mizu.Ctx) error {
	collection := c.Param("collection")
	token := extractToken(c)

	if token == "" {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Logged out successfully.",
		})
	}

	h.service.Logout(c.Context(), collection, token)

	// Clear cookie
	c.SetCookie(&http.Cookie{
		Name:     "payload-token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully.",
	})
}

// Me handles GET /api/{collection}/me
func (h *Auth) Me(c *mizu.Ctx) error {
	collection := c.Param("collection")
	token := extractToken(c)

	if token == "" {
		return c.JSON(http.StatusOK, map[string]any{
			"user": nil,
		})
	}

	user, err := h.service.Me(c.Context(), collection, token)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]any{
			"user": nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user":       user,
		"collection": collection,
	})
}

// RefreshToken handles POST /api/{collection}/refresh-token
func (h *Auth) RefreshToken(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var input struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		// Try to get from cookie
		if cookie, err := c.Request().Cookie("payload-refresh-token"); err == nil {
			input.RefreshToken = cookie.Value
		}
	}

	if input.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Refresh token required"}},
		})
	}

	result, err := h.service.RefreshToken(c.Context(), collection, input.RefreshToken)
	if err != nil {
		return authErrorResponse(c, err)
	}

	// Set new cookie
	c.SetCookie(&http.Cookie{
		Name:     "payload-token",
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]any{
		"user":         result.User,
		"token":        result.Token,
		"refreshToken": result.RefreshToken,
		"exp":          result.Exp,
		"message":      "Token refreshed successfully.",
	})
}

// Register handles POST /api/{collection}/register (if enabled)
func (h *Auth) Register(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var input auth.RegisterInput
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	result, err := h.service.Register(c.Context(), collection, &input)
	if err != nil {
		return authErrorResponse(c, err)
	}

	// Set cookie
	c.SetCookie(&http.Cookie{
		Name:     "payload-token",
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusCreated, map[string]any{
		"user":         result.User,
		"token":        result.Token,
		"refreshToken": result.RefreshToken,
		"exp":          result.Exp,
		"message":      "Account created successfully.",
	})
}

// ForgotPassword handles POST /api/{collection}/forgot-password
func (h *Auth) ForgotPassword(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var input auth.ForgotPasswordInput
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	err := h.service.ForgotPassword(c.Context(), collection, &input)
	if err != nil {
		return authErrorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "If the email exists, a password reset link has been sent.",
	})
}

// ResetPassword handles POST /api/{collection}/reset-password
func (h *Auth) ResetPassword(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var input auth.ResetPasswordInput
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	err := h.service.ResetPassword(c.Context(), collection, &input)
	if err != nil {
		return authErrorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Password reset successfully.",
	})
}

// VerifyEmail handles POST /api/{collection}/verify/{token}
func (h *Auth) VerifyEmail(c *mizu.Ctx) error {
	collection := c.Param("collection")
	token := c.Param("token")

	err := h.service.VerifyEmail(c.Context(), collection, token)
	if err != nil {
		return authErrorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Email verified successfully.",
	})
}

// Unlock handles POST /api/{collection}/unlock
func (h *Auth) Unlock(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var input struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	err := h.service.Unlock(c.Context(), collection, input.Email)
	if err != nil {
		return authErrorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account unlocked successfully.",
	})
}

// WithCollection methods return handlers with collection name pre-bound

// LoginWithCollection returns a Login handler for a specific collection.
func (h *Auth) LoginWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var input auth.LoginInput
		if err := c.BindJSON(&input, 10<<20); err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		result, err := h.service.Login(c.Context(), collection, &input)
		if err != nil {
			return authErrorResponse(c, err)
		}

		c.SetCookie(&http.Cookie{
			Name:     "payload-token",
			Value:    result.Token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		return c.JSON(http.StatusOK, map[string]any{
			"user":         result.User,
			"token":        result.Token,
			"refreshToken": result.RefreshToken,
			"exp":          result.Exp,
			"message":      "Auth Passed",
		})
	}
}

// LogoutWithCollection returns a Logout handler for a specific collection.
func (h *Auth) LogoutWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		token := extractToken(c)

		if token == "" {
			return c.JSON(http.StatusOK, map[string]string{
				"message": "Logged out successfully.",
			})
		}

		h.service.Logout(c.Context(), collection, token)

		c.SetCookie(&http.Cookie{
			Name:     "payload-token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Logged out successfully.",
		})
	}
}

// MeWithCollection returns a Me handler for a specific collection.
func (h *Auth) MeWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		token := extractToken(c)

		if token == "" {
			return c.JSON(http.StatusOK, map[string]any{
				"user": nil,
			})
		}

		user, err := h.service.Me(c.Context(), collection, token)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]any{
				"user": nil,
			})
		}

		return c.JSON(http.StatusOK, map[string]any{
			"user":       user,
			"collection": collection,
		})
	}
}

// RefreshTokenWithCollection returns a RefreshToken handler for a specific collection.
func (h *Auth) RefreshTokenWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var input struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.BindJSON(&input, 10<<20); err != nil {
			if cookie, err := c.Request().Cookie("payload-refresh-token"); err == nil {
				input.RefreshToken = cookie.Value
			}
		}

		if input.RefreshToken == "" {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Refresh token required"}},
			})
		}

		result, err := h.service.RefreshToken(c.Context(), collection, input.RefreshToken)
		if err != nil {
			return authErrorResponse(c, err)
		}

		c.SetCookie(&http.Cookie{
			Name:     "payload-token",
			Value:    result.Token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		return c.JSON(http.StatusOK, map[string]any{
			"user":         result.User,
			"token":        result.Token,
			"refreshToken": result.RefreshToken,
			"exp":          result.Exp,
			"message":      "Token refreshed successfully.",
		})
	}
}

// RegisterWithCollection returns a Register handler for a specific collection.
func (h *Auth) RegisterWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var input auth.RegisterInput
		if err := c.BindJSON(&input, 10<<20); err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		result, err := h.service.Register(c.Context(), collection, &input)
		if err != nil {
			return authErrorResponse(c, err)
		}

		c.SetCookie(&http.Cookie{
			Name:     "payload-token",
			Value:    result.Token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		return c.JSON(http.StatusCreated, map[string]any{
			"user":         result.User,
			"token":        result.Token,
			"refreshToken": result.RefreshToken,
			"exp":          result.Exp,
			"message":      "Account created successfully.",
		})
	}
}

// ForgotPasswordWithCollection returns a ForgotPassword handler for a specific collection.
func (h *Auth) ForgotPasswordWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var input auth.ForgotPasswordInput
		if err := c.BindJSON(&input, 10<<20); err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		err := h.service.ForgotPassword(c.Context(), collection, &input)
		if err != nil {
			return authErrorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "If the email exists, a password reset link has been sent.",
		})
	}
}

// ResetPasswordWithCollection returns a ResetPassword handler for a specific collection.
func (h *Auth) ResetPasswordWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var input auth.ResetPasswordInput
		if err := c.BindJSON(&input, 10<<20); err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		err := h.service.ResetPassword(c.Context(), collection, &input)
		if err != nil {
			return authErrorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Password reset successfully.",
		})
	}
}

// VerifyEmailWithCollection returns a VerifyEmail handler for a specific collection.
func (h *Auth) VerifyEmailWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		token := c.Param("token")

		err := h.service.VerifyEmail(c.Context(), collection, token)
		if err != nil {
			return authErrorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Email verified successfully.",
		})
	}
}

// UnlockWithCollection returns an Unlock handler for a specific collection.
func (h *Auth) UnlockWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var input struct {
			Email string `json:"email"`
		}
		if err := c.BindJSON(&input, 10<<20); err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		err := h.service.Unlock(c.Context(), collection, input.Email)
		if err != nil {
			return authErrorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Account unlocked successfully.",
		})
	}
}

// extractToken extracts the JWT from the request.
func extractToken(c *mizu.Ctx) string {
	// Try Authorization header first
	authHeader := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try cookie
	if cookie, err := c.Request().Cookie("payload-token"); err == nil {
		return cookie.Value
	}

	return ""
}

func authErrorResponse(c *mizu.Ctx, err error) error {
	switch err {
	case auth.ErrInvalidCredentials:
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Errors: []Error{{Message: "Invalid email or password"}},
		})
	case auth.ErrAccountLocked:
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Errors: []Error{{Message: "Account is locked. Please try again later."}},
		})
	case auth.ErrUserNotFound:
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Errors: []Error{{Message: "User not found"}},
		})
	case auth.ErrEmailExists:
		return c.JSON(http.StatusConflict, ErrorResponse{
			Errors: []Error{{Message: "Email already exists", Field: "email"}},
		})
	case auth.ErrInvalidToken:
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Errors: []Error{{Message: "Invalid token"}},
		})
	case auth.ErrTokenExpired:
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Errors: []Error{{Message: "Token expired"}},
		})
	default:
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}
}

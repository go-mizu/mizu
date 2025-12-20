package mobile

import (
	"context"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Options configures the device detection middleware.
type Options struct {
	// RequireDeviceID rejects requests without X-Device-ID header.
	// Default: false
	RequireDeviceID bool

	// RequireAppVersion rejects requests without X-App-Version header.
	// Default: false
	RequireAppVersion bool

	// OnMissingHeader is called when required headers are missing.
	// Default: returns 400 Bad Request with JSON error
	OnMissingHeader func(c *mizu.Ctx, header string) error

	// SkipUserAgent disables User-Agent parsing for platform detection.
	// Default: false (User-Agent parsing is enabled by default)
	SkipUserAgent bool
}

// New creates middleware that parses device information from headers.
// Device info is stored in request context and accessible via DeviceFromCtx.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	opts = applyDefaults(opts)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			device := parseDevice(c.Request(), opts)

			if opts.RequireDeviceID && device.DeviceID == "" {
				return handleMissing(c, opts, HeaderDeviceID)
			}
			if opts.RequireAppVersion && device.AppVersion == "" {
				return handleMissing(c, opts, HeaderAppVersion)
			}

			ctx := context.WithValue(c.Context(), deviceKey{}, device)
			*c.Request() = *c.Request().WithContext(ctx)

			return next(c)
		}
	}
}

// DeviceFromCtx extracts Device from Mizu context.
// Returns zero Device if not present.
func DeviceFromCtx(c *mizu.Ctx) Device {
	return DeviceFromContext(c.Context())
}

func applyDefaults(opts Options) Options {
	if opts.OnMissingHeader == nil {
		opts.OnMissingHeader = defaultMissingHandler
	}
	return opts
}

func handleMissing(c *mizu.Ctx, opts Options, header string) error {
	return opts.OnMissingHeader(c, header)
}

func defaultMissingHandler(c *mizu.Ctx, header string) error {
	return c.JSON(http.StatusBadRequest, map[string]any{
		"error": Error{
			Code:    ErrCodeInvalidRequest,
			Message: "Missing required header: " + header,
			Details: map[string]string{"header": header},
		},
	})
}

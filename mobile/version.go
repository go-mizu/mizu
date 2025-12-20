package mobile

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
)

// Version represents an API version.
type Version struct {
	Major int
	Minor int
}

// String returns "vN" or "vN.M" format.
func (v Version) String() string {
	if v.Minor == 0 {
		return fmt.Sprintf("v%d", v.Major)
	}
	return fmt.Sprintf("v%d.%d", v.Major, v.Minor)
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Major < other.Major {
		return -1
	}
	if v.Major > other.Major {
		return 1
	}
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}
	return 0
}

// AtLeast returns true if v >= other.
func (v Version) AtLeast(other Version) bool {
	return v.Compare(other) >= 0
}

// versionKey is the context key for API version.
type versionKey struct{}

// VersionFromContext extracts Version from request context.
func VersionFromContext(ctx context.Context) Version {
	if v, ok := ctx.Value(versionKey{}).(Version); ok {
		return v
	}
	return Version{Major: 1}
}

// VersionFromCtx extracts Version from Mizu context.
func VersionFromCtx(c *mizu.Ctx) Version {
	return VersionFromContext(c.Context())
}

// VersionOptions configures version detection middleware.
type VersionOptions struct {
	// Header is the header name for version negotiation.
	// Default: "X-API-Version"
	Header string

	// Default is the default version when none specified.
	// Default: Version{Major: 1}
	Default Version

	// Supported lists supported versions.
	// If empty, all versions are accepted.
	Supported []Version

	// Deprecated lists deprecated versions.
	// Responses include X-API-Deprecated header.
	Deprecated []Version

	// OnUnsupported handles unsupported version requests.
	// Default: returns 400 with error message
	OnUnsupported func(c *mizu.Ctx, v Version) error
}

// VersionMiddleware creates middleware that parses API version from headers.
func VersionMiddleware(opts VersionOptions) mizu.Middleware {
	opts = applyVersionDefaults(opts)

	supportedSet := make(map[string]bool)
	for _, v := range opts.Supported {
		supportedSet[v.String()] = true
	}

	deprecatedSet := make(map[string]bool)
	for _, v := range opts.Deprecated {
		deprecatedSet[v.String()] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			version := opts.Default

			// Parse version from header
			if h := c.Request().Header.Get(opts.Header); h != "" {
				if v, err := ParseVersion(h); err == nil {
					version = v
				}
			}

			// Check if version is supported
			if len(supportedSet) > 0 && !supportedSet[version.String()] {
				return opts.OnUnsupported(c, version)
			}

			// Mark deprecated versions
			if deprecatedSet[version.String()] {
				c.Header().Set("X-API-Deprecated", "true")
				c.Header().Set("X-API-Deprecated-Message", fmt.Sprintf("API version %s is deprecated", version))
			}

			ctx := context.WithValue(c.Context(), versionKey{}, version)
			*c.Request() = *c.Request().WithContext(ctx)

			return next(c)
		}
	}
}

func applyVersionDefaults(opts VersionOptions) VersionOptions {
	if opts.Header == "" {
		opts.Header = HeaderAPIVersion
	}
	if opts.Default.Major == 0 {
		opts.Default = Version{Major: 1}
	}
	if opts.OnUnsupported == nil {
		opts.OnUnsupported = defaultUnsupportedHandler
	}
	return opts
}

func defaultUnsupportedHandler(c *mizu.Ctx, v Version) error {
	return c.JSON(http.StatusBadRequest, map[string]any{
		"error": Error{
			Code:    ErrCodeInvalidRequest,
			Message: fmt.Sprintf("API version %s is not supported", v),
			Details: map[string]string{"version": v.String()},
		},
	})
}

// ParseVersion parses a version string like "v1", "v1.2", "1", "1.2".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	s = strings.TrimPrefix(s, "V")

	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 0 || parts[0] == "" {
		return Version{}, fmt.Errorf("invalid version: %q", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %w", err)
	}

	v := Version{Major: major}

	if len(parts) == 2 && parts[1] != "" {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid minor version: %w", err)
		}
		v.Minor = minor
	}

	return v, nil
}

package mobile

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
)

// Version represents a semantic API version.
type Version struct {
	Major int
	Minor int
}

// versionKey is unexported to prevent external modification.
type versionKey struct{}

// VersionFromCtx extracts Version from request context.
// Returns zero Version if not set.
func VersionFromCtx(c *mizu.Ctx) Version {
	if v, ok := c.Context().Value(versionKey{}).(Version); ok {
		return v
	}
	return Version{}
}

// ParseVersion parses version strings in various formats:
// "v1", "v1.2", "1", "1.2"
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")

	if s == "" {
		return Version{}, nil
	}

	parts := strings.SplitN(s, ".", 2)
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, err
	}

	var minor int
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return Version{}, err
		}
	}

	return Version{Major: major, Minor: minor}, nil
}

// String returns the version as "v1" or "v1.2" format.
func (v Version) String() string {
	if v.Minor == 0 {
		return "v" + strconv.Itoa(v.Major)
	}
	return "v" + strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor)
}

// IsZero returns true if version is unset.
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other.
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

// AtLeast returns true if v >= (major, minor).
func (v Version) AtLeast(major, minor int) bool {
	return v.Compare(Version{Major: major, Minor: minor}) >= 0
}

// Before returns true if v < (major, minor).
func (v Version) Before(major, minor int) bool {
	return v.Compare(Version{Major: major, Minor: minor}) < 0
}

// VersionOptions configures API version middleware.
type VersionOptions struct {
	// Header is the version header name.
	// Default: "X-API-Version"
	Header string

	// QueryParam is an alternative query parameter for version.
	// Default: "" (disabled)
	QueryParam string

	// PathPrefix enables extraction from URL path prefix (e.g., /v1/...).
	// Default: false
	PathPrefix bool

	// Default is the default version when none specified.
	// Default: Version{Major: 1}
	Default Version

	// Supported lists all supported versions.
	// Empty means no validation.
	Supported []Version

	// Deprecated lists deprecated versions (still work but warn).
	Deprecated []Version

	// OnUnsupported handles unsupported version requests.
	OnUnsupported func(c *mizu.Ctx, v Version) error

	// EchoVersion includes X-API-Version in response.
	// Default: true
	EchoVersion bool
}

// VersionMiddleware creates API versioning middleware.
func VersionMiddleware(opts VersionOptions) mizu.Middleware {
	// Apply defaults
	if opts.Header == "" {
		opts.Header = HeaderAPIVersion
	}
	if opts.Default.IsZero() {
		opts.Default = Version{Major: 1}
	}

	// Build lookup sets for O(1) checks
	supportedSet := make(map[Version]bool)
	for _, v := range opts.Supported {
		supportedSet[v] = true
	}
	deprecatedSet := make(map[Version]bool)
	for _, v := range opts.Deprecated {
		deprecatedSet[v] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var ver Version
			var err error
			var source string

			// Try header first
			if h := c.Request().Header.Get(opts.Header); h != "" {
				ver, err = ParseVersion(h)
				if err != nil {
					return SendError(c, http.StatusBadRequest, NewError(
						ErrInvalidRequest,
						"Invalid API version format",
					).WithDetails("header", opts.Header).WithDetails("value", h))
				}
				source = "header"
			}

			// Try query param
			if ver.IsZero() && opts.QueryParam != "" {
				if q := c.Query(opts.QueryParam); q != "" {
					ver, err = ParseVersion(q)
					if err != nil {
						return SendError(c, http.StatusBadRequest, NewError(
							ErrInvalidRequest,
							"Invalid API version format",
						).WithDetails("param", opts.QueryParam).WithDetails("value", q))
					}
					source = "query"
				}
			}

			// Try path prefix
			if ver.IsZero() && opts.PathPrefix {
				ver, _ = extractPathVersion(c.Request().URL.Path)
				if !ver.IsZero() {
					source = "path"
				}
			}

			// Use default if not found
			if ver.IsZero() {
				ver = opts.Default
				source = "default"
			}

			// Validate against supported versions
			if len(supportedSet) > 0 && !supportedSet[ver] {
				if opts.OnUnsupported != nil {
					return opts.OnUnsupported(c, ver)
				}
				return defaultUnsupportedVersion(c, ver, opts.Supported)
			}

			// Check deprecation
			if deprecatedSet[ver] {
				c.Header().Set(HeaderDeprecated, "true")
			}

			// Echo version in response
			if opts.EchoVersion {
				c.Header().Set(HeaderAPIVersion, ver.String())
			}

			// Store version in context
			ctx := context.WithValue(c.Context(), versionKey{}, ver)
			*c.Request() = *c.Request().WithContext(ctx)

			_ = source // suppress unused warning (useful for logging)
			return next(c)
		}
	}
}

// extractPathVersion extracts version from URL path prefix like /v1/... or /v2.0/...
func extractPathVersion(path string) (Version, string) {
	if len(path) < 2 || path[0] != '/' {
		return Version{}, path
	}

	// Find the version segment
	path = path[1:] // remove leading /
	end := strings.Index(path, "/")
	if end == -1 {
		end = len(path)
	}

	segment := path[:end]
	if len(segment) < 2 || (segment[0] != 'v' && segment[0] != 'V') {
		return Version{}, "/" + path
	}

	ver, err := ParseVersion(segment)
	if err != nil {
		return Version{}, "/" + path
	}

	// Return remaining path
	remaining := path[end:]
	if remaining == "" {
		remaining = "/"
	}
	return ver, remaining
}

func defaultUnsupportedVersion(c *mizu.Ctx, v Version, supported []Version) error {
	versions := make([]string, len(supported))
	for i, s := range supported {
		versions[i] = s.String()
	}
	return SendError(c, http.StatusBadRequest, NewError(
		ErrInvalidRequest,
		"Unsupported API version: "+v.String(),
	).WithDetails("requested", v.String()).WithDetails("supported", versions))
}

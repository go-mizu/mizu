// Package xml provides XML response handling middleware for Mizu.
package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the XML middleware.
type Options struct {
	// Indent is the indentation string for pretty printing.
	// Default: empty (no pretty printing).
	Indent string

	// Prefix is the prefix for each XML element.
	Prefix string

	// ContentType is the content type for XML responses.
	// Default: "application/xml".
	ContentType string

	// AutoParse parses XML request bodies automatically.
	// Default: false.
	AutoParse bool

	// XMLDeclaration adds XML declaration to responses.
	// Default: true.
	XMLDeclaration bool
}

// contextKey is a private type for context keys.
type contextKey string

const (
	bodyKey   contextKey = "xml_body"
	optsKey   contextKey = "xml_opts"
	formatKey contextKey = "preferred_format"
)

// New creates XML middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates XML middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.ContentType == "" {
		opts.ContentType = "application/xml"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			ctx := c.Context()

			// Auto-parse XML request body
			if opts.AutoParse && c.Request().Method != http.MethodGet {
				contentType := c.Request().Header.Get("Content-Type")
				if strings.Contains(contentType, "application/xml") ||
					strings.Contains(contentType, "text/xml") {
					body, err := io.ReadAll(c.Request().Body)
					c.Request().Body.Close()
					if err == nil {
						ctx = context.WithValue(ctx, bodyKey, body)
						c.Request().Body = io.NopCloser(bytes.NewReader(body))
					}
				}
			}

			// Check if client accepts XML
			accept := c.Request().Header.Get("Accept")
			if strings.Contains(accept, "application/xml") ||
				strings.Contains(accept, "text/xml") ||
				strings.Contains(accept, "*/*") ||
				accept == "" {
				// Add helper for XML responses
				ctx = context.WithValue(ctx, optsKey, &opts)
			}

			// Update context
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// Body returns the raw XML body from the request.
func Body(c *mizu.Ctx) []byte {
	if body, ok := c.Context().Value(bodyKey).([]byte); ok {
		return body
	}
	return nil
}

// Bind parses the XML request body into v.
func Bind(c *mizu.Ctx, v any) error {
	body := Body(c)
	if body != nil {
		return xml.Unmarshal(body, v)
	}

	// Read body directly
	data, err := io.ReadAll(c.Request().Body)
	c.Request().Body.Close()
	if err != nil {
		return err
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(data))
	return xml.Unmarshal(data, v)
}

// Response sends an XML response.
func Response(c *mizu.Ctx, status int, v any) error {
	var data []byte
	var err error

	opts, _ := c.Context().Value(optsKey).(*Options)

	if opts != nil && opts.Indent != "" {
		data, err = xml.MarshalIndent(v, opts.Prefix, opts.Indent)
	} else {
		data, err = xml.Marshal(v)
	}

	if err != nil {
		return err
	}

	contentType := "application/xml; charset=utf-8"
	if opts != nil && opts.ContentType != "" {
		contentType = opts.ContentType + "; charset=utf-8"
	}

	c.Header().Set("Content-Type", contentType)

	if opts == nil || opts.XMLDeclaration {
		c.Writer().WriteHeader(status)
		c.Writer().Write([]byte(xml.Header))
		_, err = c.Writer().Write(data)
	} else {
		c.Writer().WriteHeader(status)
		_, err = c.Writer().Write(data)
	}

	return err
}

// Error represents an XML error response.
type Error struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code"`
	Message string   `xml:"message"`
}

// SendError sends an XML error response.
func SendError(c *mizu.Ctx, status int, message string) error {
	return Response(c, status, Error{
		Code:    status,
		Message: message,
	})
}

// Pretty creates middleware that formats XML responses with indentation.
func Pretty(indent string) mizu.Middleware {
	return WithOptions(Options{Indent: indent})
}

// ContentNegotiation creates middleware that handles XML/JSON content negotiation.
func ContentNegotiation() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			accept := c.Request().Header.Get("Accept")

			// Set preferred format in context
			var format string
			if strings.Contains(accept, "application/xml") {
				format = "xml"
			} else if strings.Contains(accept, "application/json") {
				format = "json"
			} else {
				format = "json" // Default to JSON
			}

			ctx := context.WithValue(c.Context(), formatKey, format)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// PreferredFormat returns the preferred response format.
func PreferredFormat(c *mizu.Ctx) string {
	if format, ok := c.Context().Value(formatKey).(string); ok {
		return format
	}
	return "json"
}

// Respond sends a response in the preferred format (XML or JSON).
func Respond(c *mizu.Ctx, status int, v any) error {
	if PreferredFormat(c) == "xml" {
		return Response(c, status, v)
	}
	return c.JSON(status, v)
}

// Wrapper wraps data in a root XML element.
type Wrapper struct {
	XMLName xml.Name
	Data    any `xml:",innerxml"`
}

// Wrap wraps data in a named XML element.
func Wrap(name string, data any) Wrapper {
	// Pre-marshal the data to get inner XML
	inner, _ := xml.Marshal(data)
	return Wrapper{
		XMLName: xml.Name{Local: name},
		Data:    string(inner),
	}
}

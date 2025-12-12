// Package hypermedia provides HATEOAS link injection middleware for Mizu.
package hypermedia

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Link represents a hypermedia link.
type Link struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method,omitempty"`
	Title  string `json:"title,omitempty"`
	Type   string `json:"type,omitempty"`
}

// Links is a collection of links.
type Links []Link

// Options configures the hypermedia middleware.
type Options struct {
	// BaseURL is the base URL for links.
	// If empty, uses request Host.
	BaseURL string

	// LinksKey is the key for links in JSON responses.
	// Default: "_links".
	LinksKey string

	// SelfLink adds a self link automatically.
	// Default: true.
	SelfLink bool

	// LinkProvider provides links for a given path.
	LinkProvider func(path string, method string) Links
}

// contextKey is a private type for context keys.
type contextKey struct{}

// linksKey stores pending links.
var linksKey = contextKey{}

// New creates hypermedia middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{SelfLink: true})
}

// WithOptions creates hypermedia middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.LinksKey == "" {
		opts.LinksKey = "_links"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Initialize links storage
			links := make(Links, 0)

			// Add self link
			if opts.SelfLink {
				baseURL := opts.BaseURL
				if baseURL == "" {
					scheme := "http"
					if c.Request().TLS != nil {
						scheme = "https"
					}
					baseURL = scheme + "://" + c.Request().Host
				}
				links = append(links, Link{
					Href:   baseURL + c.Request().URL.Path,
					Rel:    "self",
					Method: c.Request().Method,
				})
			}

			// Get links from provider
			if opts.LinkProvider != nil {
				providerLinks := opts.LinkProvider(c.Request().URL.Path, c.Request().Method)
				links = append(links, providerLinks...)
			}

			// Store in context
			ctx := context.WithValue(c.Context(), linksKey, &links)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Capture response
			rec := &responseRecorder{
				ResponseWriter: c.Writer(),
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}
			c.SetWriter(rec)

			err := next(c)
			if err != nil {
				return err
			}

			// Restore writer
			c.SetWriter(rec.ResponseWriter)

			// Only process JSON responses
			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				// Write original response
				c.Writer().WriteHeader(rec.statusCode)
				_, _ = c.Writer().Write(rec.body.Bytes())
				return nil
			}

			// Get final links
			finalLinks, _ := c.Context().Value(linksKey).(*Links)
			if finalLinks == nil || len(*finalLinks) == 0 {
				// No links to add
				c.Writer().WriteHeader(rec.statusCode)
				_, _ = c.Writer().Write(rec.body.Bytes())
				return nil
			}

			// Parse and modify JSON
			var data map[string]any
			if err := json.Unmarshal(rec.body.Bytes(), &data); err != nil {
				// Not a JSON object, return as-is
				c.Writer().WriteHeader(rec.statusCode)
				_, _ = c.Writer().Write(rec.body.Bytes())
				return nil
			}

			// Add links
			data[opts.LinksKey] = *finalLinks

			// Re-encode
			modified, err := json.Marshal(data)
			if err != nil {
				c.Writer().WriteHeader(rec.statusCode)
				_, _ = c.Writer().Write(rec.body.Bytes())
				return nil
			}

			c.Header().Set("Content-Type", "application/json")
			c.Writer().WriteHeader(rec.statusCode)
			_, _ = c.Writer().Write(modified)
			return nil
		}
	}
}

type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
}

// AddLink adds a link to the response.
func AddLink(c *mizu.Ctx, link Link) {
	if links, ok := c.Context().Value(linksKey).(*Links); ok {
		*links = append(*links, link)
	}
}

// AddLinks adds multiple links to the response.
func AddLinks(c *mizu.Ctx, newLinks ...Link) {
	if links, ok := c.Context().Value(linksKey).(*Links); ok {
		*links = append(*links, newLinks...)
	}
}

// SetLinks replaces all links.
func SetLinks(c *mizu.Ctx, newLinks Links) {
	ctx := context.WithValue(c.Context(), linksKey, &newLinks)
	req := c.Request().WithContext(ctx)
	*c.Request() = *req
}

// GetLinks returns current links.
func GetLinks(c *mizu.Ctx) Links {
	if links, ok := c.Context().Value(linksKey).(*Links); ok {
		return *links
	}
	return nil
}

// Resource wraps data with links.
type Resource struct {
	Data  any   `json:"data"`
	Links Links `json:"_links"`
}

// NewResource creates a new resource with links.
func NewResource(data any, links ...Link) Resource {
	return Resource{
		Data:  data,
		Links: links,
	}
}

// Collection wraps a collection with pagination links.
type Collection struct {
	Items      any   `json:"items"`
	Total      int   `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	Links      Links `json:"_links"`
}

// NewCollection creates a new paginated collection.
func NewCollection(items any, total, page, pageSize int, baseURL string) Collection {
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	links := Links{
		{Href: baseURL + "?page=1", Rel: "first"},
	}

	if page > 1 {
		links = append(links, Link{
			Href: baseURL + "?page=" + itoa(page-1),
			Rel:  "prev",
		})
	}

	if page < totalPages {
		links = append(links, Link{
			Href: baseURL + "?page=" + itoa(page+1),
			Rel:  "next",
		})
	}

	links = append(links, Link{
		Href: baseURL + "?page=" + itoa(totalPages),
		Rel:  "last",
	})

	return Collection{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Links:      links,
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var result []byte
	neg := i < 0
	if neg {
		i = -i
	}

	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}

	if neg {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}

// HAL represents a HAL+JSON resource.
type HAL struct {
	Properties map[string]any   `json:"-"`
	Links      map[string]Link  `json:"_links,omitempty"`
	Embedded   map[string][]HAL `json:"_embedded,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for HAL.
func (h HAL) MarshalJSON() ([]byte, error) {
	result := make(map[string]any)

	// Add properties
	for k, v := range h.Properties {
		result[k] = v
	}

	// Add links
	if len(h.Links) > 0 {
		result["_links"] = h.Links
	}

	// Add embedded
	if len(h.Embedded) > 0 {
		result["_embedded"] = h.Embedded
	}

	return json.Marshal(result)
}

// NewHAL creates a new HAL resource.
func NewHAL(props map[string]any) *HAL {
	return &HAL{
		Properties: props,
		Links:      make(map[string]Link),
		Embedded:   make(map[string][]HAL),
	}
}

// AddLink adds a link to the HAL resource.
func (h *HAL) AddLink(rel string, link Link) *HAL {
	h.Links[rel] = link
	return h
}

// Embed adds an embedded resource.
func (h *HAL) Embed(rel string, resources ...HAL) *HAL {
	h.Embedded[rel] = append(h.Embedded[rel], resources...)
	return h
}

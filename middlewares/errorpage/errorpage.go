// Package errorpage provides custom error page middleware for Mizu.
package errorpage

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Page represents an error page configuration.
type Page struct {
	Code     int
	Title    string
	Message  string
	Template string
}

// Options configures the errorpage middleware.
type Options struct {
	// Pages defines custom error pages.
	Pages map[int]*Page

	// DefaultTemplate is the default error template.
	DefaultTemplate string

	// NotFoundHandler handles 404 errors.
	NotFoundHandler mizu.Handler

	// ErrorHandler handles other errors.
	ErrorHandler func(c *mizu.Ctx, code int) error
}

// Default error page template
const defaultTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 40px; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { font-size: 72px; margin: 0; color: #333; }
        p { color: #666; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Code}}</h1>
        <p>{{.Message}}</p>
    </div>
</body>
</html>`

// Default error pages
var defaultPages = map[int]*Page{
	400: {Code: 400, Title: "Bad Request", Message: "The request could not be understood."},
	401: {Code: 401, Title: "Unauthorized", Message: "Authentication is required."},
	403: {Code: 403, Title: "Forbidden", Message: "You don't have permission to access this resource."},
	404: {Code: 404, Title: "Not Found", Message: "The requested page could not be found."},
	405: {Code: 405, Title: "Method Not Allowed", Message: "The request method is not supported."},
	500: {Code: 500, Title: "Internal Server Error", Message: "Something went wrong on our end."},
	502: {Code: 502, Title: "Bad Gateway", Message: "The server received an invalid response."},
	503: {Code: 503, Title: "Service Unavailable", Message: "The service is temporarily unavailable."},
	504: {Code: 504, Title: "Gateway Timeout", Message: "The server took too long to respond."},
}

// New creates errorpage middleware with default pages.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates errorpage middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Pages == nil {
		opts.Pages = make(map[int]*Page)
	}
	if opts.DefaultTemplate == "" {
		opts.DefaultTemplate = defaultTemplate
	}

	// Merge with defaults
	for code, page := range defaultPages {
		if _, ok := opts.Pages[code]; !ok {
			opts.Pages[code] = page
		}
	}

	tmpl := template.Must(template.New("error").Parse(opts.DefaultTemplate))

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Wrap response writer to capture status
			rw := &statusWriter{ResponseWriter: c.Writer()}
			c.SetWriter(rw)

			err := next(c)

			// Check if we need to show an error page
			status := rw.status
			if status == 0 {
				return err
			}

			if status >= 400 && !rw.written {
				// Update context status for logging middleware
				c.Status(status)

				// Use custom handler if provided
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c, status)
				}

				// Use custom 404 handler
				if status == 404 && opts.NotFoundHandler != nil {
					return opts.NotFoundHandler(c)
				}

				// Show error page
				page := opts.Pages[status]
				if page == nil {
					page = &Page{
						Code:    status,
						Title:   http.StatusText(status),
						Message: "An error occurred.",
					}
				}

				// Use custom template if provided
				if page.Template != "" {
					customTmpl := template.Must(template.New("custom").Parse(page.Template))
					c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
					rw.ResponseWriter.WriteHeader(status)
					return customTmpl.Execute(rw.ResponseWriter, page)
				}

				c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
				rw.ResponseWriter.WriteHeader(status)
				return tmpl.Execute(rw.ResponseWriter, page)
			}

			return err
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	// Don't propagate yet - we may want to show a custom error page
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.written = true
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// NotFound returns middleware that shows 404 page.
func NotFound() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			return c.Text(http.StatusNotFound, "Not Found")
		}
	}
}

// Custom creates middleware with custom pages.
func Custom(pages map[int]*Page) mizu.Middleware {
	return WithOptions(Options{Pages: pages})
}

// Page404 creates a custom 404 page.
func Page404(title, message string) *Page {
	return &Page{Code: 404, Title: title, Message: message}
}

// Page500 creates a custom 500 page.
func Page500(title, message string) *Page {
	return &Page{Code: 500, Title: title, Message: message}
}

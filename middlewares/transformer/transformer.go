// Package transformer provides request/response transformation middleware for Mizu.
package transformer

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// RequestTransformer transforms a request.
type RequestTransformer func(r *http.Request) error

// ResponseTransformer transforms a response.
type ResponseTransformer func(statusCode int, headers http.Header, body []byte) (int, http.Header, []byte, error)

// Options configures the transformer middleware.
type Options struct {
	// RequestTransformers are applied to requests.
	RequestTransformers []RequestTransformer

	// ResponseTransformers are applied to responses.
	ResponseTransformers []ResponseTransformer
}

// New creates transformer middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates transformer middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Apply request transformers
			for _, t := range opts.RequestTransformers {
				if err := t(c.Request()); err != nil {
					return err
				}
			}

			// If no response transformers, just call next
			if len(opts.ResponseTransformers) == 0 {
				return next(c)
			}

			// Capture response
			rec := &responseRecorder{
				ResponseWriter: c.Writer(),
				body:           &bytes.Buffer{},
				headers:        make(http.Header),
			}

			// Copy existing headers
			for k, v := range c.Header() {
				rec.headers[k] = v
			}

			// Replace response writer
			originalWriter := c.Writer()
			c.SetWriter(rec)

			err := next(c)

			// Restore original writer
			c.SetWriter(originalWriter)

			if err != nil {
				return err
			}

			// Apply response transformers
			statusCode := rec.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			headers := rec.headers
			body := rec.body.Bytes()

			for _, t := range opts.ResponseTransformers {
				var transformErr error
				statusCode, headers, body, transformErr = t(statusCode, headers, body)
				if transformErr != nil {
					return transformErr
				}
			}

			// Write transformed response
			for k, v := range headers {
				for _, vv := range v {
					originalWriter.Header().Add(k, vv)
				}
			}
			originalWriter.WriteHeader(statusCode)
			_, writeErr := originalWriter.Write(body)
			return writeErr
		}
	}
}

type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
}

// Request creates middleware that applies request transformers.
func Request(transformers ...RequestTransformer) mizu.Middleware {
	return WithOptions(Options{RequestTransformers: transformers})
}

// Response creates middleware that applies response transformers.
func Response(transformers ...ResponseTransformer) mizu.Middleware {
	return WithOptions(Options{ResponseTransformers: transformers})
}

// Common request transformers

// AddHeader adds a header to the request.
func AddHeader(key, value string) RequestTransformer {
	return func(r *http.Request) error {
		r.Header.Add(key, value)
		return nil
	}
}

// SetHeader sets a header on the request.
func SetHeader(key, value string) RequestTransformer {
	return func(r *http.Request) error {
		r.Header.Set(key, value)
		return nil
	}
}

// RemoveHeader removes a header from the request.
func RemoveHeader(key string) RequestTransformer {
	return func(r *http.Request) error {
		r.Header.Del(key)
		return nil
	}
}

// RewritePath rewrites the request path.
func RewritePath(from, to string) RequestTransformer {
	return func(r *http.Request) error {
		if strings.HasPrefix(r.URL.Path, from) {
			r.URL.Path = to + strings.TrimPrefix(r.URL.Path, from)
		}
		return nil
	}
}

// AddQueryParam adds a query parameter.
func AddQueryParam(key, value string) RequestTransformer {
	return func(r *http.Request) error {
		q := r.URL.Query()
		q.Add(key, value)
		r.URL.RawQuery = q.Encode()
		return nil
	}
}

// TransformBody transforms the request body.
func TransformBody(fn func([]byte) ([]byte, error)) RequestTransformer {
	return func(r *http.Request) error {
		if r.Body == nil {
			return nil
		}

		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			return err
		}

		transformed, err := fn(body)
		if err != nil {
			return err
		}

		r.Body = io.NopCloser(bytes.NewReader(transformed))
		r.ContentLength = int64(len(transformed))
		return nil
	}
}

// Common response transformers

// AddResponseHeader adds a header to the response.
func AddResponseHeader(key, value string) ResponseTransformer {
	return func(statusCode int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		headers.Add(key, value)
		return statusCode, headers, body, nil
	}
}

// SetResponseHeader sets a header on the response.
func SetResponseHeader(key, value string) ResponseTransformer {
	return func(statusCode int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		headers.Set(key, value)
		return statusCode, headers, body, nil
	}
}

// RemoveResponseHeader removes a header from the response.
func RemoveResponseHeader(key string) ResponseTransformer {
	return func(statusCode int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		headers.Del(key)
		return statusCode, headers, body, nil
	}
}

// TransformResponseBody transforms the response body.
func TransformResponseBody(fn func([]byte) ([]byte, error)) ResponseTransformer {
	return func(statusCode int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		transformed, err := fn(body)
		if err != nil {
			return statusCode, headers, body, err
		}
		return statusCode, headers, transformed, nil
	}
}

// MapStatusCode maps a status code to another.
func MapStatusCode(from, to int) ResponseTransformer {
	return func(statusCode int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		if statusCode == from {
			return to, headers, body, nil
		}
		return statusCode, headers, body, nil
	}
}

// ReplaceBody replaces the response body based on status code.
func ReplaceBody(statusCode int, newBody []byte) ResponseTransformer {
	return func(code int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		if code == statusCode {
			return code, headers, newBody, nil
		}
		return code, headers, body, nil
	}
}

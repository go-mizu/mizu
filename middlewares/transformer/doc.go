/*
Package transformer provides request and response transformation middleware for Mizu.

The transformer middleware enables flexible modification of HTTP requests and responses,
supporting header manipulation, path rewriting, query parameter injection, and body
transformation.

# Basic Usage

Request transformation:

	app := mizu.New()
	app.Use(transformer.Request(
		transformer.AddHeader("X-API-Version", "2.0"),
		transformer.RewritePath("/api/v1", "/api/v2"),
	))

Response transformation:

	app.Use(transformer.Response(
		transformer.AddResponseHeader("X-Powered-By", "Mizu"),
		transformer.TransformResponseBody(func(body []byte) ([]byte, error) {
			return bytes.ToUpper(body), nil
		}),
	))

# Request Transformers

The package provides several built-in request transformers:

  - AddHeader: Adds a header to the request
  - SetHeader: Sets a header on the request (replaces existing)
  - RemoveHeader: Removes a header from the request
  - RewritePath: Rewrites the request URL path
  - AddQueryParam: Adds a query parameter to the request
  - TransformBody: Transforms the request body content

# Response Transformers

Built-in response transformers include:

  - AddResponseHeader: Adds a header to the response
  - SetResponseHeader: Sets a header on the response (replaces existing)
  - RemoveResponseHeader: Removes a header from the response
  - TransformResponseBody: Transforms the response body content
  - MapStatusCode: Maps one HTTP status code to another
  - ReplaceBody: Replaces response body based on status code

# Custom Transformers

Create custom request transformers:

	customReq := func(r *http.Request) error {
		// Custom request transformation logic
		r.Header.Set("X-Custom", "value")
		return nil
	}
	app.Use(transformer.Request(customReq))

Create custom response transformers:

	customResp := func(code int, headers http.Header, body []byte) (int, http.Header, []byte, error) {
		// Custom response transformation logic
		headers.Set("X-Custom", "value")
		return code, headers, body, nil
	}
	app.Use(transformer.Response(customResp))

# Advanced Configuration

Use WithOptions for combined request and response transformation:

	app.Use(transformer.WithOptions(transformer.Options{
		RequestTransformers: []transformer.RequestTransformer{
			transformer.AddHeader("X-Request-ID", uuid.New().String()),
			transformer.SetHeader("Content-Type", "application/json"),
		},
		ResponseTransformers: []transformer.ResponseTransformer{
			transformer.AddResponseHeader("X-Response-Time", time.Now().String()),
			transformer.MapStatusCode(http.StatusNotFound, http.StatusOK),
		},
	}))

# Use Cases

Common use cases include:

  - API versioning and backwards compatibility
  - Request/response header standardization
  - URL rewriting and redirection
  - Body format conversion (JSON to XML, etc.)
  - Status code normalization
  - Adding metadata to requests/responses
  - Request/response logging and debugging

# Implementation Notes

Request transformers are applied sequentially before the handler executes. Each
transformer receives the *http.Request object and can modify headers, URL, body,
and other request properties.

Response transformers use a custom responseRecorder to capture the complete response
(status code, headers, and body) before applying transformations. This allows
transformers to modify any aspect of the response before it's sent to the client.

All transformers are applied in the order they are registered. Request transformers
are executed first, followed by the handler, then response transformers.
*/
package transformer

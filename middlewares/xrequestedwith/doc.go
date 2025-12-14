/*
Package xrequestedwith provides middleware for validating the X-Requested-With HTTP header.

The X-Requested-With header is commonly used to identify AJAX-style requests.
This middleware enforces the presence and value of that header for selected
methods and paths. It can be useful as a lightweight guard for endpoints that
are intended to be called from browsers using JavaScript.

This middleware is not a substitute for authentication or CSRF protection.
Any client can set X-Requested-With. Treat it as a convention check, not a
security boundary.

# Basic Usage

Require the default "XMLHttpRequest" value for state-changing methods:

	app := mizu.New()
	app.Use(xrequestedwith.New())

By default, validation is skipped for GET, HEAD, and OPTIONS, and enforced for
other methods such as POST, PUT, PATCH, and DELETE.

# Custom Configuration

Use WithOptions to customize behavior:

	app.Use(xrequestedwith.WithOptions(xrequestedwith.Options{
		Value:       "MyCustomValue", // Require a custom header value
		SkipMethods: []string{"GET"},  // Skip only GET requests
		SkipPaths:   []string{"/webhook", "/health"}, // Skip specific paths
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "AJAX requests only",
			})
		},
	}))

# Convenience Functions

New creates middleware with default settings:

  - Value: "XMLHttpRequest"
  - SkipMethods: GET, HEAD, OPTIONS

Require validates a specific header value:

	app.Use(xrequestedwith.Require("FetchRequest"))

AJAXOnly validates all methods (no methods are skipped):

	app.Use(xrequestedwith.AJAXOnly())

# Detection Helper

IsAJAX checks whether the request has the conventional XMLHttpRequest value
(case-insensitive) without enforcing validation:

	app.Get("/data", func(c *mizu.Ctx) error {
		if xrequestedwith.IsAJAX(c) {
			return c.JSON(200, data)
		}
		return c.HTML(200, page)
	})

# Implementation Overview

Validation order:

 1. If request method is in SkipMethods, pass through
 2. If request path is in SkipPaths, pass through
 3. Compare X-Requested-With to the required value (case-insensitive)
 4. On failure, call ErrorHandler or return 400 Bad Request

Comparison uses strings.EqualFold, so values like "XMLHttpRequest",
"xmlhttprequest", and "XMLHTTPREQUEST" are treated as equivalent.

# Security Notes

Do not rely on this header as your primary CSRF defense. Prefer:

  - CSRF tokens
  - SameSite cookies
  - Origin/Referer checks where appropriate
  - Defense-in-depth

# Client Notes

Some libraries set this header automatically. With the Fetch API, set it
explicitly:

	fetch("/api", {
		method: "POST",
		headers: {
			"X-Requested-With": "XMLHttpRequest",
			"Content-Type": "application/json",
		},
		body: JSON.stringify(data),
	})

# Performance

The middleware precomputes maps for SkipMethods and SkipPaths for O(1) lookups.
The hot path is a couple of map checks and a single string comparison.
*/
package xrequestedwith

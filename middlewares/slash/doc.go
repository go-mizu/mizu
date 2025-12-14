/*
Package slash provides trailing slash handling middleware for Mizu.

The slash middleware normalizes URL paths by adding or removing trailing slashes
using HTTP redirects. This helps keep URLs consistent, avoids duplicate routes,
and improves SEO by ensuring a single canonical form for each path.

Behavior is intentionally conservative. Redirects are applied only to safe,
idempotent requests to avoid breaking clients.

# Features

  - Add or remove trailing slashes with configurable HTTP status codes
  - Redirects only GET and HEAD requests by default
  - Preserve query strings during redirection
  - Root path "/" is never modified
  - Fast string-based checks without regular expressions
  - Minimal overhead when no redirect is required

# Usage

Add trailing slashes to all URLs except root:

	app := mizu.New()
	app.Use(slash.Add())

	// /about    → 301 → /about/
	// /contact  → 301 → /contact/
	// /         → no redirect

Remove trailing slashes from all URLs except root:

	app := mizu.New()
	app.Use(slash.Remove())

	// /about/   → 301 → /about
	// /contact/ → 301 → /contact
	// /         → no redirect

Use custom HTTP status codes:

	app.Use(slash.AddCode(http.StatusFound))        // 302 Temporary redirect
	app.Use(slash.RemoveCode(http.StatusPermanentRedirect)) // 308, preserves method

# Request Methods

Redirects are only applied for:

  - GET
  - HEAD

Other methods such as POST, PUT, PATCH, and DELETE are passed through unchanged.
This avoids losing request bodies or changing semantics during redirects.

# Behavior

  - The root path "/" is never redirected
  - Query strings are always preserved
  - Default status code is 301 (Moved Permanently)
  - Any valid 3xx status code may be used
  - No redirects occur if the path is already normalized

# Performance

The middleware is designed to be lightweight:

  - Zero allocations on the fast path
  - Single string operation when building redirect targets
  - Early exit for non-matching paths and methods
  - No regular expression processing

# Best Practices

  - Choose one style, with trailing slash or without, and use it consistently
  - Prefer 301 for permanent canonical URLs
  - Apply slash middleware early in the middleware chain
  - Do not use Add and Remove together in the same application
  - Consider 308 only if you explicitly want to preserve method semantics

# Implementation Overview

For each request:

 1. Read the request method and path from c.Request()
 2. Skip processing for non-GET and non-HEAD methods
 3. Skip processing for the root path "/"
 4. Check whether the trailing slash matches the configured mode
 5. Build the redirect target if needed
 6. Append the original query string if present
 7. Issue a redirect using c.Redirect(code, target)
 8. Otherwise, call the next handler

For more information, see:
https://github.com/go-mizu/mizu/tree/main/middlewares/slash
*/
package slash

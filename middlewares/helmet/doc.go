/*
Package helmet provides security headers middleware for Mizu web applications.

The helmet middleware automatically sets security-related HTTP headers to protect
against common web vulnerabilities including clickjacking, XSS attacks, MIME type
sniffing, and other security threats.

# Overview

Helmet implements a collection of security headers following modern web security
best practices. It provides both a default configuration with recommended settings
and individual functions for fine-grained control over specific headers.

# Quick Start

The simplest way to use helmet is with the Default() function which applies
recommended security headers:

	app := mizu.New()
	app.Use(helmet.Default())

This sets the following headers:
  - X-Content-Type-Options: nosniff
  - X-Frame-Options: SAMEORIGIN
  - X-DNS-Prefetch-Control: off
  - X-Download-Options: noopen
  - X-Permitted-Cross-Domain-Policies: none
  - Referrer-Policy: strict-origin-when-cross-origin
  - Cross-Origin-Opener-Policy: same-origin
  - Cross-Origin-Resource-Policy: same-origin
  - Origin-Agent-Cluster: ?1

# Custom Configuration

For custom security requirements, use the New() function with an Options struct:

	app.Use(helmet.New(helmet.Options{
		ContentSecurityPolicy: "default-src 'self'",
		XFrameOptions:         "DENY",
		XContentTypeOptions:   true,
		ReferrerPolicy:        "no-referrer",
		StrictTransportSecurity: &helmet.HSTSOptions{
			MaxAge:            365 * 24 * time.Hour,
			IncludeSubDomains: true,
			Preload:           true,
		},
	}))

# Individual Headers

You can also apply individual security headers using dedicated functions:

	app.Use(helmet.ContentSecurityPolicy("default-src 'self'"))
	app.Use(helmet.XFrameOptions("DENY"))
	app.Use(helmet.StrictTransportSecurity(365*24*time.Hour, true, true))

# Security Headers Reference

Content-Security-Policy (CSP)
Controls which resources the browser is allowed to load. Prevents XSS and
injection attacks by restricting script sources, styles, images, and other
resource types.

X-Frame-Options
Prevents clickjacking attacks by controlling whether the page can be
embedded in frames or iframes. Values: DENY, SAMEORIGIN.

X-Content-Type-Options
Prevents MIME type sniffing by forcing browsers to respect the declared
Content-Type header. Always set to "nosniff".

Referrer-Policy
Controls how much referrer information is included with requests.
Prevents information leakage through the Referer header.

Strict-Transport-Security (HSTS)
Forces browsers to use HTTPS for all future requests to the domain.
Protects against protocol downgrade attacks and cookie hijacking.

Permissions-Policy
Controls which browser features and APIs can be used. Replaces the
deprecated Feature-Policy header.

Cross-Origin-Opener-Policy (COOP)
Isolates the browsing context from other origins, protecting against
cross-origin attacks like Spectre.

Cross-Origin-Embedder-Policy (COEP)
Prevents a document from loading cross-origin resources that don't
explicitly grant permission.

Cross-Origin-Resource-Policy (CORP)
Protects resources from being loaded by other origins, preventing
cross-origin attacks.

Origin-Agent-Cluster
Requests that the browser place the origin in a separate agent cluster,
improving isolation between origins.

X-DNS-Prefetch-Control
Controls DNS prefetching. When disabled (off), prevents privacy leakage
through DNS requests.

X-Download-Options
Prevents IE from executing downloads in the site's context. Set to "noopen".

X-Permitted-Cross-Domain-Policies
Controls cross-domain access for Adobe Flash and PDF documents.
Recommended value: "none".

# HSTS Configuration

HTTP Strict Transport Security (HSTS) requires special configuration through
the HSTSOptions struct:

	&helmet.HSTSOptions{
		MaxAge:            365 * 24 * time.Hour,  // Duration to remember HTTPS
		IncludeSubDomains: true,                  // Apply to all subdomains
		Preload:           true,                  // Eligible for browser preload lists
	}

Warning: HSTS is powerful but can lock users out if misconfigured. Only enable
preload after ensuring your entire domain and all subdomains support HTTPS.

# Implementation Details

The middleware is implemented as a higher-order function that wraps request
handlers. Security headers are injected before passing control to the next
handler in the chain.

Header setting is conditional:
  - String fields: Set only if non-empty
  - Boolean fields: Set only if true
  - Pointer fields: Set only if non-nil

This ensures headers are only sent when explicitly configured, preventing
unwanted or empty headers.

# Performance

The helmet middleware has minimal performance impact:
  - No dynamic allocations during request processing
  - All configuration computed once at middleware creation
  - Only simple string operations and header setting per request

# Best Practices

1. Start with Default() and customize as needed
2. Test security headers thoroughly before deploying to production
3. Use Content-Security-Policy in report-only mode initially
4. Enable HSTS only after confirming HTTPS works correctly
5. Review browser console for CSP violations during development

# Example: Complete Security Setup

	app := mizu.New()

	// Apply comprehensive security headers
	app.Use(helmet.New(helmet.Options{
		// Strict CSP for XSS protection
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' cdn.example.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' fonts.gstatic.com",

		// Prevent clickjacking
		XFrameOptions: "DENY",

		// Enable all recommended options
		XContentTypeOptions: true,

		// Minimal referrer information
		ReferrerPolicy: "no-referrer",

		// HSTS with subdomains and preload
		StrictTransportSecurity: &helmet.HSTSOptions{
			MaxAge:            365 * 24 * time.Hour,
			IncludeSubDomains: true,
			Preload:           true,
		},

		// Disable dangerous browser features
		PermissionsPolicy: "geolocation=(), microphone=(), camera=()",

		// Cross-origin isolation
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginResourcePolicy: "same-origin",

		// Origin isolation
		OriginAgentCluster: true,

		// Privacy and security for legacy headers
		XDownloadOptions:              true,
		XPermittedCrossDomainPolicies: "none",
	}))

For more information and examples, see:
https://github.com/go-mizu/mizu/tree/main/middlewares/helmet
*/
package helmet

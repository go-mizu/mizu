package frontend

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// assetType classifies assets for caching purposes.
type assetType int

const (
	assetHashed   assetType = iota // app.a1b2c3d4.js
	assetUnhashed                  // logo.png
	assetHTML                      // index.html
	assetMap                       // app.js.map
)

// hashPattern matches content hashes in filenames.
// Matches: app.a1b2c3d4.js, vendor-abc123.css, chunk.ABC123.js
var hashPattern = regexp.MustCompile(`[._-][a-fA-F0-9]{6,}$`)

// classifyAsset determines the asset type for caching.
func classifyAsset(path string) assetType {
	ext := strings.ToLower(filepath.Ext(path))

	// HTML files
	if ext == ".html" || ext == ".htm" {
		return assetHTML
	}

	// Source maps
	if ext == ".map" {
		return assetMap
	}

	// Check for content hash in filename
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ext)

	if hashPattern.MatchString(name) {
		return assetHashed
	}

	return assetUnhashed
}

// setCacheHeaders sets appropriate cache headers based on asset type.
func setCacheHeaders(w http.ResponseWriter, path string, cfg CacheConfig) {
	// Check custom patterns first
	for pattern, duration := range cfg.Patterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			setCacheControl(w, duration, false)
			return
		}
	}

	// Default caching by asset type
	switch classifyAsset(path) {
	case assetHashed:
		setCacheControl(w, cfg.HashedAssets, true)
	case assetUnhashed:
		setCacheControl(w, cfg.UnhashedAssets, false)
	case assetHTML:
		setHTMLCacheHeaders(w, cfg)
	case assetMap:
		// No cache for source maps
		w.Header().Set("Cache-Control", "no-cache")
	}
}

// setHTMLCacheHeaders sets cache headers for HTML files.
func setHTMLCacheHeaders(w http.ResponseWriter, cfg CacheConfig) {
	if cfg.HTML > 0 {
		setCacheControl(w, cfg.HTML, false)
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}

// setCacheControl sets Cache-Control header with appropriate directives.
func setCacheControl(w http.ResponseWriter, duration time.Duration, immutable bool) {
	seconds := int(duration.Seconds())
	if seconds <= 0 {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}

	var b strings.Builder
	b.WriteString("public, max-age=")
	b.WriteString(strconv.Itoa(seconds))

	if immutable {
		b.WriteString(", immutable")
	}

	w.Header().Set("Cache-Control", b.String())
}

package mobile

import (
	"encoding/json"
	"strings"

	"github.com/go-mizu/mizu"
)

// DeepLink represents a deep link configuration.
type DeepLink struct {
	// Scheme is the custom URL scheme (e.g., "myapp")
	Scheme string

	// Host is the universal link domain (e.g., "example.com")
	Host string

	// Paths are allowed deep link paths
	Paths []string

	// Fallback is the web fallback URL
	Fallback string
}

// AppleAppConfig is the iOS app configuration for apple-app-site-association.
type AppleAppConfig struct {
	// TeamID is the Apple Developer Team ID
	TeamID string

	// BundleID is the iOS app bundle identifier
	BundleID string

	// Paths are the URL paths to handle (supports * wildcards)
	Paths []string
}

// AndroidAppConfig is the Android app configuration for assetlinks.json.
type AndroidAppConfig struct {
	// PackageName is the Android app package name
	PackageName string

	// Fingerprints are SHA256 certificate fingerprints
	Fingerprints []string

	// Paths are the URL paths to handle
	Paths []string
}

// UniversalLinkConfig combines iOS and Android configurations.
type UniversalLinkConfig struct {
	// Apple contains iOS app configurations
	Apple []AppleAppConfig

	// Android contains Android app configurations
	Android []AndroidAppConfig

	// WebCredentials allows sharing credentials with apps
	WebCredentials []string

	// Fallback is the web fallback URL
	Fallback string
}

// AppleAppSiteAssociation generates apple-app-site-association JSON.
func (d DeepLink) AppleAppSiteAssociation(teamID, bundleID string) []byte {
	paths := d.Paths
	if len(paths) == 0 {
		paths = []string{"*"}
	}

	aasa := map[string]any{
		"applinks": map[string]any{
			"apps": []string{}, // Must be empty array per Apple spec
			"details": []map[string]any{
				{
					"appID": teamID + "." + bundleID,
					"paths": paths,
				},
			},
		},
	}

	b, _ := json.MarshalIndent(aasa, "", "  ")
	return b
}

// AppleAppSiteAssociationV2 generates the newer format with components.
func AppleAppSiteAssociationV2(configs []AppleAppConfig) []byte {
	details := make([]map[string]any, 0, len(configs))
	for _, cfg := range configs {
		paths := cfg.Paths
		if len(paths) == 0 {
			paths = []string{"*"}
		}

		// Convert paths to components format
		components := make([]map[string]string, 0, len(paths))
		for _, path := range paths {
			components = append(components, map[string]string{
				"/": path,
			})
		}

		details = append(details, map[string]any{
			"appIDs":     []string{cfg.TeamID + "." + cfg.BundleID},
			"components": components,
		})
	}

	aasa := map[string]any{
		"applinks": map[string]any{
			"details": details,
		},
	}

	b, _ := json.MarshalIndent(aasa, "", "  ")
	return b
}

// AssetLinks generates .well-known/assetlinks.json for Android.
func (d DeepLink) AssetLinks(packageName, fingerprint string) []byte {
	links := []map[string]any{
		{
			"relation": []string{"delegate_permission/common.handle_all_urls"},
			"target": map[string]any{
				"namespace":                "android_app",
				"package_name":             packageName,
				"sha256_cert_fingerprints": []string{fingerprint},
			},
		},
	}

	b, _ := json.MarshalIndent(links, "", "  ")
	return b
}

// AssetLinksMultiple generates assetlinks.json for multiple apps.
func AssetLinksMultiple(configs []AndroidAppConfig) []byte {
	links := make([]map[string]any, 0)

	for _, cfg := range configs {
		links = append(links, map[string]any{
			"relation": []string{"delegate_permission/common.handle_all_urls"},
			"target": map[string]any{
				"namespace":                "android_app",
				"package_name":             cfg.PackageName,
				"sha256_cert_fingerprints": cfg.Fingerprints,
			},
		})
	}

	b, _ := json.MarshalIndent(links, "", "  ")
	return b
}

// UniversalLinkMiddleware creates middleware that serves deep link verification files.
func UniversalLinkMiddleware(cfg UniversalLinkConfig) mizu.Middleware {
	// Pre-generate responses
	var aasa, assetLinks []byte

	if len(cfg.Apple) > 0 {
		aasa = AppleAppSiteAssociationV2(cfg.Apple)
	}

	if len(cfg.Android) > 0 {
		assetLinks = AssetLinksMultiple(cfg.Android)
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			// Apple App Site Association
			if path == "/.well-known/apple-app-site-association" ||
				path == "/apple-app-site-association" {
				if len(aasa) == 0 {
					return next(c)
				}
				c.Header().Set("Content-Type", "application/json")
				return c.Bytes(200, aasa, "application/json")
			}

			// Android Asset Links
			if path == "/.well-known/assetlinks.json" {
				if len(assetLinks) == 0 {
					return next(c)
				}
				c.Header().Set("Content-Type", "application/json")
				return c.Bytes(200, assetLinks, "application/json")
			}

			return next(c)
		}
	}
}

// DeepLinkMiddleware creates middleware for a simple deep link configuration.
func DeepLinkMiddleware(link DeepLink, teamID, bundleID, packageName, fingerprint string) mizu.Middleware {
	return UniversalLinkMiddleware(UniversalLinkConfig{
		Apple: []AppleAppConfig{
			{TeamID: teamID, BundleID: bundleID, Paths: link.Paths},
		},
		Android: []AndroidAppConfig{
			{PackageName: packageName, Fingerprints: []string{fingerprint}},
		},
		Fallback: link.Fallback,
	})
}

// DeepLinkHandler creates a handler that redirects to app or web fallback.
// Uses smart detection based on User-Agent.
func DeepLinkHandler(scheme, fallbackURL string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		device := DeviceFromCtx(c)

		// Build the deep link URL
		deepLinkURL := scheme + "://" + strings.TrimPrefix(c.Request().URL.Path, "/")
		if c.Request().URL.RawQuery != "" {
			deepLinkURL += "?" + c.Request().URL.RawQuery
		}

		// For mobile platforms, return HTML with both deep link and fallback
		if device != nil && device.Platform.IsMobile() {
			return c.HTML(200, renderDeepLinkHTML(deepLinkURL, fallbackURL))
		}

		// For web/desktop, redirect to fallback
		return c.Redirect(302, fallbackURL)
	}
}

// renderDeepLinkHTML creates HTML that attempts deep link then falls back.
func renderDeepLinkHTML(deepLink, fallback string) string {
	return `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Opening App...</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; text-align: center; padding: 50px; }
.loader { margin: 20px auto; width: 40px; height: 40px; border: 4px solid #f3f3f3;
  border-top: 4px solid #333; border-radius: 50%; animation: spin 1s linear infinite; }
@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
a { color: #007aff; text-decoration: none; }
</style>
</head>
<body>
<div class="loader"></div>
<p>Opening app...</p>
<p>If the app doesn't open, <a href="` + fallback + `">click here</a></p>
<script>
(function() {
  var timeout = setTimeout(function() { window.location = "` + fallback + `"; }, 2500);
  window.location = "` + deepLink + `";
  window.addEventListener('blur', function() { clearTimeout(timeout); });
})();
</script>
</body>
</html>`
}

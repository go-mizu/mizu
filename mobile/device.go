package mobile

import (
	"context"
	"net/http"
	"regexp"
	"strings"
)

// Device represents a mobile device making a request.
// Extracted from User-Agent and custom headers.
type Device struct {
	// Platform is the OS: "ios", "android", "web", or "unknown"
	Platform Platform `json:"platform"`

	// Version is the OS version (e.g., "17.2", "14.0")
	Version string `json:"version,omitempty"`

	// AppVersion is the client app version from X-App-Version header
	AppVersion string `json:"app_version,omitempty"`

	// AppBuild is the client app build number from X-App-Build header
	AppBuild string `json:"app_build,omitempty"`

	// DeviceID is a unique device identifier from X-Device-ID header
	DeviceID string `json:"device_id,omitempty"`

	// DeviceModel is the device model from X-Device-Model header (e.g., "iPhone15,2")
	DeviceModel string `json:"device_model,omitempty"`

	// Locale is the device locale from Accept-Language header
	Locale string `json:"locale,omitempty"`

	// Timezone is the device timezone from X-Timezone header
	Timezone string `json:"timezone,omitempty"`

	// PushToken is the push notification token from X-Push-Token header
	PushToken string `json:"push_token,omitempty"`
}

// Platform represents the mobile operating system.
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformWeb     Platform = "web"
	PlatformUnknown Platform = "unknown"
)

// Is checks if platform matches.
func (p Platform) Is(other Platform) bool { return p == other }

// IsMobile returns true if platform is iOS or Android.
func (p Platform) IsMobile() bool { return p == PlatformIOS || p == PlatformAndroid }

// String returns the platform as a string.
func (p Platform) String() string { return string(p) }

// deviceKey is the context key for device information.
type deviceKey struct{}

// DeviceFromContext extracts Device from request context.
// Returns zero Device if not present.
func DeviceFromContext(ctx context.Context) Device {
	if d, ok := ctx.Value(deviceKey{}).(Device); ok {
		return d
	}
	return Device{}
}

// Header names for mobile device information.
const (
	HeaderDeviceID    = "X-Device-ID"
	HeaderAppVersion  = "X-App-Version"
	HeaderAppBuild    = "X-App-Build"
	HeaderDeviceModel = "X-Device-Model"
	HeaderTimezone    = "X-Timezone"
	HeaderPushToken   = "X-Push-Token"   //nolint:gosec // header name, not a credential
	HeaderAPIVersion  = "X-API-Version"
	HeaderSyncToken   = "X-Sync-Token"   //nolint:gosec // header name, not a credential
)

// parseDevice extracts device information from request headers.
func parseDevice(r *http.Request, opts Options) Device {
	d := Device{
		DeviceID:    r.Header.Get(HeaderDeviceID),
		AppVersion:  r.Header.Get(HeaderAppVersion),
		AppBuild:    r.Header.Get(HeaderAppBuild),
		DeviceModel: r.Header.Get(HeaderDeviceModel),
		Timezone:    r.Header.Get(HeaderTimezone),
		PushToken:   r.Header.Get(HeaderPushToken),
		Platform:    PlatformUnknown,
	}

	// Parse Accept-Language for locale
	if lang := r.Header.Get("Accept-Language"); lang != "" {
		d.Locale = parseLocale(lang)
	}

	// Parse User-Agent for platform detection (enabled by default)
	if !opts.SkipUserAgent {
		ua := r.Header.Get("User-Agent")
		d.Platform, d.Version = parseUserAgent(ua)
	}

	return d
}

// parseLocale extracts primary locale from Accept-Language header.
func parseLocale(lang string) string {
	// Accept-Language: en-US,en;q=0.9,es;q=0.8
	if idx := strings.Index(lang, ","); idx != -1 {
		lang = lang[:idx]
	}
	if idx := strings.Index(lang, ";"); idx != -1 {
		lang = lang[:idx]
	}
	return strings.TrimSpace(lang)
}

// User-Agent patterns for platform detection.
var (
	iosPattern     = regexp.MustCompile(`(?i)(iphone|ipad|ipod).*?os\s*([\d_.]+)`)
	androidPattern = regexp.MustCompile(`(?i)android\s*([\d.]+)`)
	webViewIOS     = regexp.MustCompile(`(?i)darwin/`)
	webViewAndroid = regexp.MustCompile(`(?i)wv\)`)
)

// parseUserAgent extracts platform and version from User-Agent string.
func parseUserAgent(ua string) (Platform, string) {
	if ua == "" {
		return PlatformUnknown, ""
	}

	// Check for iOS
	if matches := iosPattern.FindStringSubmatch(ua); len(matches) >= 3 {
		version := strings.ReplaceAll(matches[2], "_", ".")
		return PlatformIOS, version
	}

	// Check for Android
	if matches := androidPattern.FindStringSubmatch(ua); len(matches) >= 2 {
		return PlatformAndroid, matches[1]
	}

	// Check for iOS WebView (Darwin-based)
	if webViewIOS.MatchString(ua) {
		return PlatformIOS, ""
	}

	// Check for Android WebView
	if webViewAndroid.MatchString(ua) {
		return PlatformAndroid, ""
	}

	// Check for common browser patterns (web platform)
	if strings.Contains(strings.ToLower(ua), "mozilla") ||
		strings.Contains(strings.ToLower(ua), "chrome") ||
		strings.Contains(strings.ToLower(ua), "safari") ||
		strings.Contains(strings.ToLower(ua), "firefox") {
		return PlatformWeb, ""
	}

	return PlatformUnknown, ""
}

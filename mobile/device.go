package mobile

import (
	"net/http"
	"regexp"
	"strings"
)

// Platform represents a mobile operating system.
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformWindows Platform = "windows"
	PlatformMacOS   Platform = "macos"
	PlatformWeb     Platform = "web"
	PlatformUnknown Platform = "unknown"
)

// String returns the platform as a string.
func (p Platform) String() string { return string(p) }

// IsMobile returns true for mobile platforms.
func (p Platform) IsMobile() bool {
	return p == PlatformIOS || p == PlatformAndroid
}

// IsDesktop returns true for desktop platforms.
func (p Platform) IsDesktop() bool {
	return p == PlatformMacOS || p == PlatformWindows
}

// IsNative returns true for native app platforms (not web).
func (p Platform) IsNative() bool {
	return p != PlatformWeb && p != PlatformUnknown
}

// Device contains information about the client device.
type Device struct {
	// Platform is the operating system (ios, android, windows, etc.)
	Platform Platform

	// OSVersion is the OS version (e.g., "17.0", "14.0")
	OSVersion string

	// AppVersion is the client app version (e.g., "1.2.3")
	AppVersion string

	// AppBuild is the build number (e.g., "123", "2024.01.15")
	AppBuild string

	// DeviceID is a unique device identifier
	DeviceID string

	// DeviceModel is the device model (e.g., "iPhone15,2", "Pixel 8")
	DeviceModel string

	// Locale is the device locale (e.g., "en-US", "ja-JP")
	Locale string

	// Timezone is the IANA timezone (e.g., "America/New_York")
	Timezone string

	// PushToken is the push notification token (if provided)
	PushToken string

	// PushProvider is APNS, FCM, or WNS
	PushProvider PushProvider

	// UserAgent is the raw User-Agent header
	UserAgent string
}

// User-Agent parsing patterns
var (
	iosPattern     = regexp.MustCompile(`(?i)(?:iPhone|iPad|iPod).*OS\s+([\d_]+)`)
	androidPattern = regexp.MustCompile(`(?i)Android\s+([\d.]+)`)
	windowsPattern = regexp.MustCompile(`(?i)Windows\s+(?:NT\s+)?([\d.]+)`)
	macPattern     = regexp.MustCompile(`(?i)Mac\s+OS\s+X\s+([\d_]+)`)
)

// parseDevice extracts device information from HTTP request.
func parseDevice(r *http.Request, opts Options) *Device {
	d := &Device{
		Platform:    PlatformUnknown,
		DeviceID:    r.Header.Get(HeaderDeviceID),
		AppVersion:  r.Header.Get(HeaderAppVersion),
		AppBuild:    r.Header.Get(HeaderAppBuild),
		DeviceModel: r.Header.Get(HeaderDeviceModel),
		Timezone:    r.Header.Get(HeaderTimezone),
		Locale:      r.Header.Get(HeaderLocale),
		PushToken:   r.Header.Get(HeaderPushToken),
		UserAgent:   r.Header.Get("User-Agent"),
	}

	// Parse platform from header first (explicit override)
	if platform := r.Header.Get(HeaderPlatform); platform != "" {
		d.Platform = parsePlatformString(platform)
	}

	// Parse OS version from header
	if osVersion := r.Header.Get(HeaderOSVersion); osVersion != "" {
		d.OSVersion = osVersion
	}

	// Parse User-Agent if platform not set and not skipped
	if d.Platform == PlatformUnknown && !opts.SkipUserAgent {
		d.Platform, d.OSVersion = parseUserAgent(d.UserAgent)
	}

	// Infer push provider from platform
	if d.PushToken != "" {
		d.PushProvider = inferPushProvider(d.Platform)
	}

	return d
}

// parsePlatformString converts a platform string to Platform.
func parsePlatformString(s string) Platform {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ios", "iphone", "ipad":
		return PlatformIOS
	case "android":
		return PlatformAndroid
	case "windows":
		return PlatformWindows
	case "macos", "mac", "osx":
		return PlatformMacOS
	case "web":
		return PlatformWeb
	default:
		return PlatformUnknown
	}
}

// parseUserAgent extracts platform and OS version from User-Agent.
func parseUserAgent(ua string) (Platform, string) {
	if ua == "" {
		return PlatformUnknown, ""
	}

	// Check iOS first (more specific pattern)
	if matches := iosPattern.FindStringSubmatch(ua); len(matches) > 1 {
		version := strings.ReplaceAll(matches[1], "_", ".")
		return PlatformIOS, version
	}

	// Check Android
	if matches := androidPattern.FindStringSubmatch(ua); len(matches) > 1 {
		return PlatformAndroid, matches[1]
	}

	// Check Windows
	if matches := windowsPattern.FindStringSubmatch(ua); len(matches) > 1 {
		return PlatformWindows, matches[1]
	}

	// Check macOS
	if matches := macPattern.FindStringSubmatch(ua); len(matches) > 1 {
		version := strings.ReplaceAll(matches[1], "_", ".")
		return PlatformMacOS, version
	}

	// Default to web for browser-like User-Agents
	if strings.Contains(strings.ToLower(ua), "mozilla") ||
		strings.Contains(strings.ToLower(ua), "chrome") ||
		strings.Contains(strings.ToLower(ua), "safari") {
		return PlatformWeb, ""
	}

	return PlatformUnknown, ""
}

// inferPushProvider determines push provider from platform.
func inferPushProvider(p Platform) PushProvider {
	switch p {
	case PlatformIOS, PlatformMacOS:
		return PushAPNS
	case PlatformAndroid:
		return PushFCM
	case PlatformWindows:
		return PushWNS
	default:
		return ""
	}
}

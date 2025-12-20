package mobile

import (
	"context"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// AppInfo contains app store information.
type AppInfo struct {
	// CurrentVersion is the latest version in the store
	CurrentVersion string `json:"current_version"`

	// MinimumVersion is the minimum required version
	MinimumVersion string `json:"minimum_version"`

	// UpdateURL is the store URL for updating
	UpdateURL string `json:"update_url"`

	// ReleaseNotes is the latest release notes
	ReleaseNotes string `json:"release_notes,omitempty"`

	// ReleasedAt is when the current version was released
	ReleasedAt time.Time `json:"released_at,omitempty"`

	// ForceUpdate indicates if update is mandatory
	ForceUpdate bool `json:"force_update"`

	// MaintenanceMode indicates if the service is under maintenance
	MaintenanceMode bool `json:"maintenance_mode"`

	// MaintenanceMessage is the message to display during maintenance
	MaintenanceMessage string `json:"maintenance_message,omitempty"`

	// MaintenanceEndTime is when maintenance is expected to end
	MaintenanceEndTime *time.Time `json:"maintenance_end_time,omitempty"`

	// Features lists enabled feature flags
	Features map[string]bool `json:"features,omitempty"`
}

// AppInfoProvider fetches app info from a backend.
type AppInfoProvider interface {
	GetAppInfo(ctx context.Context, platform Platform, bundleID string) (*AppInfo, error)
}

// UpdateStatus represents the update check result.
type UpdateStatus struct {
	// Available indicates an update is available
	Available bool `json:"update_available"`

	// Required indicates the update is mandatory
	Required bool `json:"update_required"`

	// CurrentVersion is the client's current version
	CurrentVersion string `json:"current_version"`

	// LatestVersion is the latest store version
	LatestVersion string `json:"latest_version"`

	// MinimumVersion is the minimum required version
	MinimumVersion string `json:"minimum_version,omitempty"`

	// UpdateURL is the store URL
	UpdateURL string `json:"update_url,omitempty"`

	// ReleaseNotes for the latest version
	ReleaseNotes string `json:"release_notes,omitempty"`
}

// CheckUpdate compares client version against store versions.
func CheckUpdate(clientVersion, latestVersion, minimumVersion string) UpdateStatus {
	status := UpdateStatus{
		CurrentVersion: clientVersion,
		LatestVersion:  latestVersion,
		MinimumVersion: minimumVersion,
	}

	// Check if update is available
	if compareVersions(clientVersion, latestVersion) < 0 {
		status.Available = true
	}

	// Check if update is required
	if minimumVersion != "" && compareVersions(clientVersion, minimumVersion) < 0 {
		status.Required = true
	}

	return status
}

// AppInfoHandler creates an endpoint for app version checking.
func AppInfoHandler(provider AppInfoProvider) mizu.Handler {
	return func(c *mizu.Ctx) error {
		device := DeviceFromCtx(c)
		if device == nil {
			return SendError(c, http.StatusBadRequest, NewError(
				ErrInvalidRequest,
				"Device context not available",
			))
		}

		// Get bundle ID from query or header
		bundleID := c.Query("bundle_id")
		if bundleID == "" {
			bundleID = c.Request().Header.Get("X-Bundle-ID")
		}

		info, err := provider.GetAppInfo(c.Context(), device.Platform, bundleID)
		if err != nil {
			return SendError(c, http.StatusInternalServerError, NewError(
				ErrInternal,
				"Failed to fetch app info",
			))
		}

		if info == nil {
			return SendError(c, http.StatusNotFound, NewError(
				ErrNotFound,
				"App info not found",
			))
		}

		// Check for maintenance mode
		if info.MaintenanceMode {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{
				"maintenance":    true,
				"message":        info.MaintenanceMessage,
				"end_time":       info.MaintenanceEndTime,
				"minimum_version": info.MinimumVersion,
			})
		}

		// Build update status
		status := CheckUpdate(device.AppVersion, info.CurrentVersion, info.MinimumVersion)
		status.UpdateURL = info.UpdateURL
		status.ReleaseNotes = info.ReleaseNotes

		// Set response headers
		c.Header().Set(HeaderMinVersion, info.MinimumVersion)

		return c.JSON(http.StatusOK, status)
	}
}

// StaticAppInfo is a simple AppInfoProvider with static values.
type StaticAppInfo struct {
	// Apps maps platform -> bundle ID -> AppInfo
	Apps map[Platform]map[string]*AppInfo

	// Default is used when no specific match is found
	Default *AppInfo
}

// GetAppInfo implements AppInfoProvider.
func (s *StaticAppInfo) GetAppInfo(_ context.Context, platform Platform, bundleID string) (*AppInfo, error) {
	if s.Apps != nil {
		if platformApps, ok := s.Apps[platform]; ok {
			if info, ok := platformApps[bundleID]; ok {
				return info, nil
			}
			// Try wildcard bundle ID
			if info, ok := platformApps["*"]; ok {
				return info, nil
			}
		}
	}
	return s.Default, nil
}

// NewStaticAppInfo creates a StaticAppInfo with defaults.
func NewStaticAppInfo(currentVersion, minVersion, updateURL string) *StaticAppInfo {
	return &StaticAppInfo{
		Default: &AppInfo{
			CurrentVersion: currentVersion,
			MinimumVersion: minVersion,
			UpdateURL:      updateURL,
		},
	}
}

// WithPlatformApp adds platform-specific app info.
func (s *StaticAppInfo) WithPlatformApp(platform Platform, bundleID string, info *AppInfo) *StaticAppInfo {
	if s.Apps == nil {
		s.Apps = make(map[Platform]map[string]*AppInfo)
	}
	if s.Apps[platform] == nil {
		s.Apps[platform] = make(map[string]*AppInfo)
	}
	s.Apps[platform][bundleID] = info
	return s
}

// StoreURLs contains app store URLs by platform.
type StoreURLs struct {
	iOS     string `json:"ios,omitempty"`
	Android string `json:"android,omitempty"`
	Windows string `json:"windows,omitempty"`
	Web     string `json:"web,omitempty"`
}

// URLFor returns the store URL for a platform.
func (u StoreURLs) URLFor(platform Platform) string {
	switch platform {
	case PlatformIOS:
		return u.iOS
	case PlatformAndroid:
		return u.Android
	case PlatformWindows:
		return u.Windows
	default:
		return u.Web
	}
}

// NewIOSStoreURL creates an iOS App Store URL.
func NewIOSStoreURL(appID string) string {
	return "https://apps.apple.com/app/id" + appID
}

// NewAndroidStoreURL creates a Google Play Store URL.
func NewAndroidStoreURL(packageName string) string {
	return "https://play.google.com/store/apps/details?id=" + packageName
}

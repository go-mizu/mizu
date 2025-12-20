package mobile

import (
	"regexp"
	"time"

	"github.com/go-mizu/mizu"
)

// PushProvider is the push notification service.
type PushProvider string

const (
	PushAPNS PushProvider = "apns" // Apple Push Notification Service
	PushFCM  PushProvider = "fcm"  // Firebase Cloud Messaging
	PushWNS  PushProvider = "wns"  // Windows Notification Service
)

// String returns the provider as a string.
func (p PushProvider) String() string { return string(p) }

// PushToken represents a device push token.
type PushToken struct {
	// Token is the push token value
	Token string `json:"token"`

	// Provider is the push service (apns, fcm, wns)
	Provider PushProvider `json:"provider"`

	// DeviceID is the associated device identifier
	DeviceID string `json:"device_id,omitempty"`

	// Sandbox indicates APNS sandbox environment
	Sandbox bool `json:"sandbox,omitempty"`

	// CreatedAt is when the token was registered
	CreatedAt time.Time `json:"created_at,omitempty"`

	// UpdatedAt is when the token was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// AppVersion is the app version when token was registered
	AppVersion string `json:"app_version,omitempty"`
}

// Token validation patterns
var (
	// APNS tokens are 64 hex characters (32 bytes)
	apnsPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

	// FCM tokens are typically 140-200 characters, alphanumeric with : and -
	fcmPattern = regexp.MustCompile(`^[a-zA-Z0-9_:.-]{100,300}$`)
)

// ValidateAPNS validates APNS token format.
func ValidateAPNS(token string) bool {
	return apnsPattern.MatchString(token)
}

// ValidateFCM validates FCM token format.
func ValidateFCM(token string) bool {
	return fcmPattern.MatchString(token)
}

// ValidateToken validates a push token based on provider.
func ValidateToken(token string, provider PushProvider) bool {
	switch provider {
	case PushAPNS:
		return ValidateAPNS(token)
	case PushFCM:
		return ValidateFCM(token)
	case PushWNS:
		// WNS tokens are URIs, basic validation
		return len(token) > 0 && len(token) < 500
	default:
		return len(token) > 0
	}
}

// ParsePushToken extracts push token from request.
// Infers provider from device platform if not specified.
func ParsePushToken(c *mizu.Ctx) *PushToken {
	token := c.Request().Header.Get(HeaderPushToken)
	if token == "" {
		return nil
	}

	pt := &PushToken{
		Token:     token,
		DeviceID:  c.Request().Header.Get(HeaderDeviceID),
		CreatedAt: time.Now(),
	}

	// Infer provider from device context
	if device := DeviceFromCtx(c); device != nil {
		pt.Provider = device.PushProvider
		pt.AppVersion = device.AppVersion
	}

	// Auto-detect provider from token format if not set
	if pt.Provider == "" {
		pt.Provider = detectPushProvider(token)
	}

	return pt
}

// detectPushProvider attempts to detect provider from token format.
func detectPushProvider(token string) PushProvider {
	if ValidateAPNS(token) {
		return PushAPNS
	}
	if ValidateFCM(token) {
		return PushFCM
	}
	return ""
}

// PushRegistration represents a push token registration request.
type PushRegistration struct {
	Token    string       `json:"token"`
	Provider PushProvider `json:"provider,omitempty"`
	Sandbox  bool         `json:"sandbox,omitempty"`
	Topics   []string     `json:"topics,omitempty"` // Subscription topics
}

// Validate validates the push registration.
func (r *PushRegistration) Validate() error {
	if r.Token == "" {
		return NewError(ErrValidation, "push token is required")
	}

	if r.Provider != "" && !ValidateToken(r.Token, r.Provider) {
		return NewError(ErrValidation, "invalid push token format").
			WithDetails("provider", r.Provider.String())
	}

	return nil
}

// PushPayload is a cross-platform push notification payload.
type PushPayload struct {
	// Title is the notification title
	Title string `json:"title,omitempty"`

	// Body is the notification body text
	Body string `json:"body,omitempty"`

	// Badge is the app badge count (iOS)
	Badge *int `json:"badge,omitempty"`

	// Sound is the notification sound
	Sound string `json:"sound,omitempty"`

	// Data is custom payload data
	Data map[string]any `json:"data,omitempty"`

	// Category is the notification category (for actions)
	Category string `json:"category,omitempty"`

	// ThreadID groups notifications (iOS)
	ThreadID string `json:"thread_id,omitempty"`

	// ChannelID is the notification channel (Android)
	ChannelID string `json:"channel_id,omitempty"`

	// CollapseKey groups notifications (Android/FCM)
	CollapseKey string `json:"collapse_key,omitempty"`

	// Priority is the notification priority
	Priority string `json:"priority,omitempty"`

	// TTL is the time-to-live in seconds
	TTL int `json:"ttl,omitempty"`

	// ContentAvailable triggers background fetch (iOS)
	ContentAvailable bool `json:"content_available,omitempty"`

	// MutableContent allows notification modification (iOS)
	MutableContent bool `json:"mutable_content,omitempty"`
}

// WithData adds a data field to the payload.
func (p *PushPayload) WithData(key string, value any) *PushPayload {
	if p.Data == nil {
		p.Data = make(map[string]any)
	}
	p.Data[key] = value
	return p
}

// SetBadge sets the badge count.
func (p *PushPayload) SetBadge(n int) *PushPayload {
	p.Badge = &n
	return p
}

// ToAPNS converts to APNS payload format.
func (p *PushPayload) ToAPNS() map[string]any {
	aps := map[string]any{}

	alert := map[string]any{}
	if p.Title != "" {
		alert["title"] = p.Title
	}
	if p.Body != "" {
		alert["body"] = p.Body
	}
	if len(alert) > 0 {
		aps["alert"] = alert
	}

	if p.Badge != nil {
		aps["badge"] = *p.Badge
	}
	if p.Sound != "" {
		aps["sound"] = p.Sound
	}
	if p.Category != "" {
		aps["category"] = p.Category
	}
	if p.ThreadID != "" {
		aps["thread-id"] = p.ThreadID
	}
	if p.ContentAvailable {
		aps["content-available"] = 1
	}
	if p.MutableContent {
		aps["mutable-content"] = 1
	}

	payload := map[string]any{"aps": aps}

	// Add custom data
	for k, v := range p.Data {
		payload[k] = v
	}

	return payload
}

// ToFCM converts to FCM payload format.
func (p *PushPayload) ToFCM() map[string]any {
	message := map[string]any{}

	notification := map[string]any{}
	if p.Title != "" {
		notification["title"] = p.Title
	}
	if p.Body != "" {
		notification["body"] = p.Body
	}
	if len(notification) > 0 {
		message["notification"] = notification
	}

	// Android-specific options
	android := map[string]any{}
	if p.ChannelID != "" {
		if androidNotification, ok := android["notification"].(map[string]any); ok {
			androidNotification["channel_id"] = p.ChannelID
		} else {
			android["notification"] = map[string]any{"channel_id": p.ChannelID}
		}
	}
	if p.CollapseKey != "" {
		android["collapse_key"] = p.CollapseKey
	}
	if p.Priority != "" {
		android["priority"] = p.Priority
	}
	if p.TTL > 0 {
		android["ttl"] = p.TTL
	}
	if len(android) > 0 {
		message["android"] = android
	}

	// Add custom data
	if len(p.Data) > 0 {
		data := make(map[string]string)
		for k, v := range p.Data {
			if s, ok := v.(string); ok {
				data[k] = s
			}
		}
		if len(data) > 0 {
			message["data"] = data
		}
	}

	return message
}

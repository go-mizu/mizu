package mobile

import (
	"time"

	"github.com/go-mizu/mizu"
)

// TokenType represents push notification service.
type TokenType string

const (
	TokenAPNS TokenType = "apns" // Apple Push Notification Service
	TokenFCM  TokenType = "fcm"  // Firebase Cloud Messaging
)

// PushToken represents a registered push notification token.
type PushToken struct {
	Token     string    `json:"token"`
	Type      TokenType `json:"type"`
	DeviceID  string    `json:"device_id"`
	Platform  Platform  `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
}

// IsValid returns true if the push token is non-empty.
func (p PushToken) IsValid() bool {
	return p.Token != ""
}

// ParsePushToken extracts push token from request headers.
func ParsePushToken(c *mizu.Ctx) PushToken {
	device := DeviceFromCtx(c)

	token := c.Request().Header.Get(HeaderPushToken)
	if token == "" {
		return PushToken{}
	}

	tokenType := TokenFCM
	if device.Platform == PlatformIOS {
		tokenType = TokenAPNS
	}

	return PushToken{
		Token:     token,
		Type:      tokenType,
		DeviceID:  device.DeviceID,
		Platform:  device.Platform,
		CreatedAt: time.Now(),
	}
}

// PushTokenRequest represents a push token registration request body.
type PushTokenRequest struct {
	Token    string    `json:"token"`
	Type     TokenType `json:"type,omitempty"`
	DeviceID string    `json:"device_id,omitempty"`
}

// ToPushToken converts request to PushToken with context from device.
func (r PushTokenRequest) ToPushToken(c *mizu.Ctx) PushToken {
	device := DeviceFromCtx(c)

	tokenType := r.Type
	if tokenType == "" {
		tokenType = TokenFCM
		if device.Platform == PlatformIOS {
			tokenType = TokenAPNS
		}
	}

	deviceID := r.DeviceID
	if deviceID == "" {
		deviceID = device.DeviceID
	}

	return PushToken{
		Token:     r.Token,
		Type:      tokenType,
		DeviceID:  deviceID,
		Platform:  device.Platform,
		CreatedAt: time.Now(),
	}
}

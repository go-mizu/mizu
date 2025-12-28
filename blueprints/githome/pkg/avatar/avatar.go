package avatar

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// GravatarURL returns a Gravatar URL for an email address
func GravatarURL(email string, size int) string {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.Sum([]byte(email))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=%d&d=identicon", hex.EncodeToString(hash[:]), size)
}

// DefaultURL returns a default avatar URL based on username
func DefaultURL(username string, size int) string {
	// Use UI Avatars service for text-based avatars
	return fmt.Sprintf("https://ui-avatars.com/api/?name=%s&size=%d&background=random", username, size)
}

// GetURL returns the avatar URL, falling back to defaults
func GetURL(avatarURL, email, username string, size int) string {
	if avatarURL != "" {
		return avatarURL
	}
	if email != "" {
		return GravatarURL(email, size)
	}
	return DefaultURL(username, size)
}

package discord

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	BaseAPI = "https://discord.com/api/v10"
	BaseURL = "https://discord.com"

	EntityGuild       = "guild"
	EntityChannel     = "channel"
	EntityMessagePage = "message_page"
	EntityUser        = "user"

	// Channel types
	ChannelTypeGuildText  = 0
	ChannelTypeGuildNews  = 5
	ChannelTypeGuildForum = 15
)

// ParsedRef is a normalized reference to a Discord entity.
type ParsedRef struct {
	EntityType string
	ID         string
	// For message_page: channel ID
	ChannelID string
	// For message_page: before cursor (may be empty = fetch latest)
	Before string
	// Canonical queue URL
	URL string
}

// Guild represents a Discord server.
type Guild struct {
	GuildID                    string    `json:"guild_id"`
	Name                       string    `json:"name"`
	Description                string    `json:"description"`
	IconURL                    string    `json:"icon_url"`
	MemberCount                int64     `json:"member_count"`
	ApproximatePresenceCount   int64     `json:"approximate_presence_count"`
	OwnerID                    string    `json:"owner_id"`
	FeaturesJSON               string    `json:"features_json"`
	FetchedAt                  time.Time `json:"fetched_at"`
}

// Channel represents a Discord channel.
type Channel struct {
	ChannelID     string    `json:"channel_id"`
	GuildID       string    `json:"guild_id"`
	Name          string    `json:"name"`
	ChannelType   int       `json:"channel_type"`
	Topic         string    `json:"topic"`
	Position      int       `json:"position"`
	ParentID      string    `json:"parent_id"`
	NSFW          bool      `json:"nsfw"`
	LastMessageID string    `json:"last_message_id"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// Message represents a Discord message.
type Message struct {
	MessageID          string    `json:"message_id"`
	ChannelID          string    `json:"channel_id"`
	GuildID            string    `json:"guild_id"`
	AuthorID           string    `json:"author_id"`
	AuthorUsername     string    `json:"author_username"`
	Content            string    `json:"content"`
	Timestamp          time.Time `json:"timestamp"`
	EditedTimestamp    time.Time `json:"edited_timestamp"`
	MessageType        int       `json:"message_type"`
	Pinned             bool      `json:"pinned"`
	MentionEveryone    bool      `json:"mention_everyone"`
	AttachmentsJSON    string    `json:"attachments_json"`
	EmbedsJSON         string    `json:"embeds_json"`
	ReactionsJSON      string    `json:"reactions_json"`
	ReferencedMessageID string   `json:"referenced_message_id"`
	FetchedAt          time.Time `json:"fetched_at"`
}

// User represents a Discord user profile.
type User struct {
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	GlobalName    string    `json:"global_name"`
	Discriminator string    `json:"discriminator"`
	AvatarURL     string    `json:"avatar_url"`
	Bot           bool      `json:"bot"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// QueueItem is a pending work item in the crawl queue.
type QueueItem struct {
	ID         int64
	URL        string
	EntityType string
	Priority   int
}

// DBStats holds row counts for the main database.
type DBStats struct {
	Guilds   int64
	Channels int64
	Messages int64
	Users    int64
	DBSize   int64
}

// ParseRef normalizes any supported input form to a ParsedRef.
// For entity types guild/channel/user: input may be a raw snowflake ID or Discord URL.
// For message_page: input is a queue URL like discord://channels/{id}/messages?before={id}
func ParseRef(raw, expected string) (ParsedRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedRef{}, fmt.Errorf("empty input")
	}

	// Internal queue URL scheme
	if strings.HasPrefix(raw, "discord://") {
		return parseDiscordScheme(raw, expected)
	}

	// https://discord.com/channels/{guild_id}/{channel_id}
	if strings.HasPrefix(raw, "https://discord.com/channels/") {
		parts := strings.Split(strings.TrimPrefix(raw, "https://discord.com/channels/"), "/")
		if len(parts) >= 2 && parts[1] != "" {
			if expected != "" && expected != EntityChannel {
				return ParsedRef{}, fmt.Errorf("expected %s, got channel from URL", expected)
			}
			id := parts[1]
			return ParsedRef{
				EntityType: EntityChannel,
				ID:         id,
				URL:        channelQueueURL(id),
			}, nil
		}
		if len(parts) >= 1 && parts[0] != "" {
			if expected != "" && expected != EntityGuild {
				return ParsedRef{}, fmt.Errorf("expected %s, got guild from URL", expected)
			}
			id := parts[0]
			return ParsedRef{
				EntityType: EntityGuild,
				ID:         id,
				URL:        guildQueueURL(id),
			}, nil
		}
	}

	// Assume raw snowflake ID
	if !isSnowflake(raw) {
		return ParsedRef{}, fmt.Errorf("cannot parse discord entity from %q", raw)
	}
	if expected == "" {
		return ParsedRef{}, fmt.Errorf("entity type required for bare snowflake ID %q", raw)
	}
	return ParsedRef{
		EntityType: expected,
		ID:         raw,
		URL:        entityQueueURL(expected, raw),
	}, nil
}

func parseDiscordScheme(raw, expected string) (ParsedRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return ParsedRef{}, fmt.Errorf("parse discord URL %q: %w", raw, err)
	}
	// discord://guilds/{id}
	// discord://channels/{id}
	// discord://users/{id}
	// discord://channels/{channel_id}/messages?before={id}
	path := strings.Trim(u.Host+u.Path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return ParsedRef{}, fmt.Errorf("invalid discord scheme URL %q", raw)
	}
	kind := parts[0]
	id := parts[1]

	switch kind {
	case "guilds":
		return ParsedRef{EntityType: EntityGuild, ID: id, URL: raw}, nil
	case "channels":
		if len(parts) == 3 && parts[2] == "messages" {
			before := u.Query().Get("before")
			return ParsedRef{
				EntityType: EntityMessagePage,
				ChannelID:  id,
				Before:     before,
				URL:        raw,
			}, nil
		}
		return ParsedRef{EntityType: EntityChannel, ID: id, URL: raw}, nil
	case "users":
		return ParsedRef{EntityType: EntityUser, ID: id, URL: raw}, nil
	default:
		return ParsedRef{}, fmt.Errorf("unknown discord entity kind %q in %q", kind, raw)
	}
}

func isSnowflake(s string) bool {
	if len(s) < 15 || len(s) > 20 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GuildQueueURL returns the canonical queue URL for a guild.
func GuildQueueURL(id string) string {
	return "discord://guilds/" + id
}

func guildQueueURL(id string) string { return GuildQueueURL(id) }

// ChannelQueueURL returns the canonical queue URL for a channel.
func ChannelQueueURL(id string) string {
	return "discord://channels/" + id
}

func channelQueueURL(id string) string { return ChannelQueueURL(id) }

// UserQueueURL returns the canonical queue URL for a user.
func UserQueueURL(id string) string {
	return "discord://users/" + id
}

func userQueueURL(id string) string { return UserQueueURL(id) }

// MessagePageQueueURL returns the canonical queue URL for a message page.
func MessagePageQueueURL(channelID, before string) string {
	u := "discord://channels/" + channelID + "/messages"
	if before != "" {
		u += "?before=" + before
	}
	return u
}

func messagePageQueueURL(channelID, before string) string {
	return MessagePageQueueURL(channelID, before)
}

func entityQueueURL(entityType, id string) string {
	switch entityType {
	case EntityGuild:
		return GuildQueueURL(id)
	case EntityChannel:
		return ChannelQueueURL(id)
	case EntityUser:
		return UserQueueURL(id)
	default:
		return "discord://" + entityType + "/" + id
	}
}

// GuildName extracts the name from a raw guild API object.
func GuildName(m map[string]any) string {
	name, _ := m["name"].(string)
	return name
}

func avatarURL(userID, hash string) string {
	if hash == "" || userID == "" {
		return ""
	}
	ext := "png"
	if strings.HasPrefix(hash, "a_") {
		ext = "gif"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", userID, hash, ext)
}

func iconURL(guildID, hash string) string {
	if hash == "" || guildID == "" {
		return ""
	}
	ext := "png"
	if strings.HasPrefix(hash, "a_") {
		ext = "gif"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.%s", guildID, hash, ext)
}

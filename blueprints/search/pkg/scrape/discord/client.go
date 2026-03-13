package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client is a Discord REST API v10 client with bucket-aware rate limiting.
type Client struct {
	http    *http.Client
	token   string
	delay   time.Duration
	rl      *rateLimiter
	mu      sync.Mutex
	lastReq time.Time
}

// NewClient creates a new Discord API client.
func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		token: cfg.Token,
		delay: cfg.Delay,
		rl:    newRateLimiter(),
	}
}

// -- Rate limiter ----------------------------------------------------------

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	globalMu sync.Mutex
	globalUntil time.Time
}

type bucket struct {
	mu        sync.Mutex
	remaining int
	resetAt   time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: make(map[string]*bucket)}
}

func (rl *rateLimiter) bucket(key string) *bucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{remaining: 1}
		rl.buckets[key] = b
	}
	return b
}

func (rl *rateLimiter) waitGlobal(ctx context.Context) error {
	rl.globalMu.Lock()
	until := rl.globalUntil
	rl.globalMu.Unlock()
	if wait := time.Until(until); wait > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil
}

func (rl *rateLimiter) setGlobal(retryAfter float64) {
	rl.globalMu.Lock()
	rl.globalUntil = time.Now().Add(time.Duration(retryAfter * float64(time.Second)))
	rl.globalMu.Unlock()
}

func (rl *rateLimiter) waitBucket(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	b := rl.bucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.remaining <= 0 {
		wait := time.Until(b.resetAt)
		if wait > 0 {
			b.mu.Unlock()
			select {
			case <-ctx.Done():
				b.mu.Lock()
				return ctx.Err()
			case <-time.After(wait):
			}
			b.mu.Lock()
		}
	}
	if b.remaining > 0 {
		b.remaining--
	}
	return nil
}

func (rl *rateLimiter) update(bucketKey string, remaining int, resetAfter float64) {
	if bucketKey == "" {
		return
	}
	b := rl.bucket(bucketKey)
	b.mu.Lock()
	b.remaining = remaining
	b.resetAt = time.Now().Add(time.Duration(resetAfter * float64(time.Second)))
	b.mu.Unlock()
}

// -- HTTP helpers ----------------------------------------------------------

func (c *Client) globalDelay() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.delay <= 0 {
		return
	}
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

// routeKey returns a stable key for rate-limit tracking: "METHOD /path/template".
// We strip numeric IDs so different entity fetches share the same bucket.
func routeKey(method, path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if isSnowflake(p) {
			parts[i] = "{id}"
		}
	}
	return method + " " + strings.Join(parts, "/")
}

// do performs an authenticated GET request to the Discord API.
// It handles rate limits, retries on 429, and returns the response body.
func (c *Client) do(ctx context.Context, method, path string) ([]byte, int, error) {
	rKey := routeKey(method, path)

	for attempt := 0; attempt < 5; attempt++ {
		if err := c.rl.waitGlobal(ctx); err != nil {
			return nil, 0, err
		}
		if err := c.rl.waitBucket(ctx, rKey); err != nil {
			return nil, 0, err
		}
		c.globalDelay()

		rawURL := BaseAPI + path
		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", c.token)
		req.Header.Set("User-Agent", "DiscordBot (https://github.com/go-mizu, 1.0)")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, 0, err
		}

		// Parse rate limit headers
		bucketID := resp.Header.Get("X-RateLimit-Bucket")
		remaining, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
		resetAfterStr := resp.Header.Get("X-RateLimit-Reset-After")
		resetAfter, _ := strconv.ParseFloat(resetAfterStr, 64)
		if bucketID != "" {
			c.rl.update(bucketID, remaining, resetAfter)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
		resp.Body.Close()
		if err != nil {
			return nil, resp.StatusCode, err
		}

		if resp.StatusCode == 429 {
			var rlBody struct {
				RetryAfter float64 `json:"retry_after"`
				Global     bool    `json:"global"`
			}
			_ = json.Unmarshal(body, &rlBody)
			wait := rlBody.RetryAfter
			if wait <= 0 {
				wait = 1.0
			}
			if rlBody.Global {
				c.rl.setGlobal(wait)
			}
			select {
			case <-ctx.Done():
				return nil, resp.StatusCode, ctx.Err()
			case <-time.After(time.Duration(wait * float64(time.Second))):
			}
			continue
		}

		return body, resp.StatusCode, nil
	}
	return nil, 429, fmt.Errorf("rate limited after 5 attempts on %s %s", method, path)
}

// -- API methods -----------------------------------------------------------

// FetchMe fetches the authenticated user's profile.
func (c *Client) FetchMe(ctx context.Context) (map[string]any, int, error) {
	return c.fetchJSON(ctx, "/users/@me")
}

// FetchGuilds fetches the list of guilds the authenticated user is in.
func (c *Client) FetchGuilds(ctx context.Context) ([]map[string]any, int, error) {
	body, code, err := c.do(ctx, "GET", "/users/@me/guilds?limit=200")
	if err != nil {
		return nil, code, err
	}
	if code != 200 {
		return nil, code, fmt.Errorf("GET /users/@me/guilds: HTTP %d", code)
	}
	var guilds []map[string]any
	if err := json.Unmarshal(body, &guilds); err != nil {
		return nil, code, fmt.Errorf("decode guilds: %w", err)
	}
	return guilds, code, nil
}

// FetchGuild fetches a guild by ID (with_counts=true for member/presence counts).
func (c *Client) FetchGuild(ctx context.Context, guildID string) (map[string]any, int, error) {
	return c.fetchJSON(ctx, "/guilds/"+guildID+"?with_counts=true")
}

// FetchGuildChannels fetches all channels in a guild.
func (c *Client) FetchGuildChannels(ctx context.Context, guildID string) ([]map[string]any, int, error) {
	body, code, err := c.do(ctx, "GET", "/guilds/"+guildID+"/channels")
	if err != nil {
		return nil, code, err
	}
	if code != 200 {
		return nil, code, fmt.Errorf("GET /guilds/%s/channels: HTTP %d", guildID, code)
	}
	var channels []map[string]any
	if err := json.Unmarshal(body, &channels); err != nil {
		return nil, code, fmt.Errorf("decode channels: %w", err)
	}
	return channels, code, nil
}

// FetchChannel fetches a single channel by ID.
func (c *Client) FetchChannel(ctx context.Context, channelID string) (map[string]any, int, error) {
	return c.fetchJSON(ctx, "/channels/"+channelID)
}

// FetchMessages fetches up to 100 messages from a channel.
// before is a snowflake ID for pagination (empty = fetch latest).
func (c *Client) FetchMessages(ctx context.Context, channelID, before string) ([]map[string]any, int, error) {
	path := "/channels/" + channelID + "/messages?limit=100"
	if before != "" {
		path += "&before=" + before
	}
	body, code, err := c.do(ctx, "GET", path)
	if err != nil {
		return nil, code, err
	}
	if code == 403 {
		return nil, code, nil // No access — not an error, just skip
	}
	if code != 200 {
		return nil, code, fmt.Errorf("GET /channels/%s/messages: HTTP %d", channelID, code)
	}
	var messages []map[string]any
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, code, fmt.Errorf("decode messages: %w", err)
	}
	return messages, code, nil
}

// FetchUser fetches a user profile by ID.
func (c *Client) FetchUser(ctx context.Context, userID string) (map[string]any, int, error) {
	return c.fetchJSON(ctx, "/users/"+userID)
}

func (c *Client) fetchJSON(ctx context.Context, path string) (map[string]any, int, error) {
	body, code, err := c.do(ctx, "GET", path)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	if code != 200 {
		return nil, code, fmt.Errorf("GET %s: HTTP %d: %s", path, code, truncate(string(body), 200))
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, code, fmt.Errorf("decode %s: %w", path, err)
	}
	return m, code, nil
}

// -- JSON helpers ----------------------------------------------------------

func getString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return strings.TrimSpace(v)
}

func getBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func marshalJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" || string(b) == "[]" {
		return ""
	}
	return string(b)
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ParseGuild converts a raw API guild object to a Guild struct.
func ParseGuild(m map[string]any) Guild {
	id := getString(m, "id")
	iconHash := getString(m, "icon")
	return Guild{
		GuildID:                  id,
		Name:                     getString(m, "name"),
		Description:              getString(m, "description"),
		IconURL:                  iconURL(id, iconHash),
		MemberCount:              getInt64(m, "member_count") + getInt64(m, "approximate_member_count"),
		ApproximatePresenceCount: getInt64(m, "approximate_presence_count"),
		OwnerID:                  getString(m, "owner_id"),
		FeaturesJSON:             marshalJSON(m["features"]),
		FetchedAt:                time.Now(),
	}
}

// ParseChannel converts a raw API channel object to a Channel struct.
func ParseChannel(m map[string]any, guildID string) Channel {
	gid := getString(m, "guild_id")
	if gid == "" {
		gid = guildID
	}
	return Channel{
		ChannelID:     getString(m, "id"),
		GuildID:       gid,
		Name:          getString(m, "name"),
		ChannelType:   getInt(m, "type"),
		Topic:         getString(m, "topic"),
		Position:      getInt(m, "position"),
		ParentID:      getString(m, "parent_id"),
		NSFW:          getBool(m, "nsfw"),
		LastMessageID: getString(m, "last_message_id"),
		FetchedAt:     time.Now(),
	}
}

// ParseMessage converts a raw API message object to a Message struct.
func ParseMessage(m map[string]any, channelID, guildID string) Message {
	cid := getString(m, "channel_id")
	if cid == "" {
		cid = channelID
	}
	gid := getString(m, "guild_id")
	if gid == "" {
		gid = guildID
	}

	authorID := ""
	authorUsername := ""
	if author, ok := m["author"].(map[string]any); ok {
		authorID = getString(author, "id")
		authorUsername = getString(author, "global_name")
		if authorUsername == "" {
			authorUsername = getString(author, "username")
		}
	}

	refMsgID := ""
	if ref, ok := m["message_reference"].(map[string]any); ok {
		refMsgID = getString(ref, "message_id")
	}

	return Message{
		MessageID:           getString(m, "id"),
		ChannelID:           cid,
		GuildID:             gid,
		AuthorID:            authorID,
		AuthorUsername:      authorUsername,
		Content:             getString(m, "content"),
		Timestamp:           parseTimestamp(getString(m, "timestamp")),
		EditedTimestamp:     parseTimestamp(getString(m, "edited_timestamp")),
		MessageType:         getInt(m, "type"),
		Pinned:              getBool(m, "pinned"),
		MentionEveryone:     getBool(m, "mention_everyone"),
		AttachmentsJSON:     marshalJSON(m["attachments"]),
		EmbedsJSON:          marshalJSON(m["embeds"]),
		ReactionsJSON:       marshalJSON(m["reactions"]),
		ReferencedMessageID: refMsgID,
		FetchedAt:           time.Now(),
	}
}

// ParseUser converts a raw API user object to a User struct.
func ParseUser(m map[string]any) User {
	id := getString(m, "id")
	avatarHash := getString(m, "avatar")
	return User{
		UserID:        id,
		Username:      getString(m, "username"),
		GlobalName:    getString(m, "global_name"),
		Discriminator: getString(m, "discriminator"),
		AvatarURL:     avatarURL(id, avatarHash),
		Bot:           getBool(m, "bot"),
		FetchedAt:     time.Now(),
	}
}

// isTextChannel returns true if the channel type is a crawlable text channel.
func isTextChannel(chType int) bool {
	return chType == ChannelTypeGuildText || chType == ChannelTypeGuildNews || chType == ChannelTypeGuildForum
}

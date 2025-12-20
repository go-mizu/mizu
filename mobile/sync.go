package mobile

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// SyncToken is an opaque token representing sync state.
// Encodes a timestamp for delta synchronization.
type SyncToken string

// NewSyncToken creates a token from timestamp.
func NewSyncToken(t time.Time) SyncToken {
	// Encode Unix timestamp in nanoseconds as base64
	nano := t.UnixNano()
	return SyncToken(base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(nano, 36)),
	))
}

// Time extracts timestamp from token.
// Returns zero time if token is invalid.
func (t SyncToken) Time() time.Time {
	if t == "" {
		return time.Time{}
	}

	decoded, err := base64.RawURLEncoding.DecodeString(string(t))
	if err != nil {
		return time.Time{}
	}

	nano, err := strconv.ParseInt(string(decoded), 36, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(0, nano)
}

// String returns the token as a string.
func (t SyncToken) String() string {
	return string(t)
}

// IsEmpty returns true if token is empty.
func (t SyncToken) IsEmpty() bool {
	return t == ""
}

// SyncRequest represents a sync request from client.
type SyncRequest struct {
	// Token is the last sync token (empty for initial sync)
	Token SyncToken

	// Resources lists specific resources to sync (empty for all)
	Resources []string

	// FullSync forces a complete resync
	FullSync bool

	// Limit is the maximum number of items to return
	Limit int
}

// Since returns the time to sync from.
// Returns zero time for initial sync.
func (r SyncRequest) Since() time.Time {
	if r.FullSync || r.Token.IsEmpty() {
		return time.Time{}
	}
	return r.Token.Time()
}

// IsInitial returns true for initial sync (no token or full sync).
func (r SyncRequest) IsInitial() bool {
	return r.FullSync || r.Token.IsEmpty()
}

// ParseSyncRequest extracts sync request from headers and query params.
func ParseSyncRequest(c *mizu.Ctx) SyncRequest {
	req := SyncRequest{
		Limit: 100, // Default limit
	}

	// Try header first
	if token := c.Request().Header.Get(HeaderSyncToken); token != "" {
		req.Token = SyncToken(token)
	}

	// Fall back to query param
	if req.Token.IsEmpty() {
		if token := c.Query("sync_token"); token != "" {
			req.Token = SyncToken(token)
		}
	}

	// Check for full sync flag
	if full := c.Query("full_sync"); full == "true" || full == "1" {
		req.FullSync = true
	}

	// Parse resources
	if resources := c.Query("resources"); resources != "" {
		req.Resources = strings.Split(resources, ",")
		for i := range req.Resources {
			req.Resources[i] = strings.TrimSpace(req.Resources[i])
		}
	}

	// Parse limit
	if limit := c.Query("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			req.Limit = n
			if req.Limit > 1000 {
				req.Limit = 1000 // Cap at 1000
			}
		}
	}

	return req
}

// SyncResponse wraps data with sync metadata.
type SyncResponse[T any] struct {
	Data      T         `json:"data"`
	SyncToken SyncToken `json:"sync_token"`
	HasMore   bool      `json:"has_more"`
	FullSync  bool      `json:"full_sync,omitempty"`
}

// NewSyncResponse creates a sync response.
func NewSyncResponse[T any](data T, token SyncToken, hasMore bool) SyncResponse[T] {
	return SyncResponse[T]{
		Data:      data,
		SyncToken: token,
		HasMore:   hasMore,
	}
}

// Delta represents changes since last sync.
type Delta[T any] struct {
	Created []T      `json:"created,omitempty"`
	Updated []T      `json:"updated,omitempty"`
	Deleted []string `json:"deleted,omitempty"` // IDs of deleted items
}

// IsEmpty returns true if delta has no changes.
func (d Delta[T]) IsEmpty() bool {
	return len(d.Created) == 0 && len(d.Updated) == 0 && len(d.Deleted) == 0
}

// Count returns total number of changes.
func (d Delta[T]) Count() int {
	return len(d.Created) + len(d.Updated) + len(d.Deleted)
}

// SyncDelta wraps a delta with sync metadata.
type SyncDelta[T any] struct {
	Delta[T]
	SyncToken SyncToken `json:"sync_token"`
	HasMore   bool      `json:"has_more"`
	FullSync  bool      `json:"full_sync,omitempty"`
}

// NewSyncDelta creates a sync delta response.
func NewSyncDelta[T any](delta Delta[T], token SyncToken, hasMore bool) SyncDelta[T] {
	return SyncDelta[T]{
		Delta:     delta,
		SyncToken: token,
		HasMore:   hasMore,
	}
}

// NewFullSyncDelta creates a sync delta response for full sync.
func NewFullSyncDelta[T any](delta Delta[T], token SyncToken, hasMore bool) SyncDelta[T] {
	return SyncDelta[T]{
		Delta:     delta,
		SyncToken: token,
		HasMore:   hasMore,
		FullSync:  true,
	}
}

// SetSyncToken sets the sync token response header.
func SetSyncToken(c *mizu.Ctx, token SyncToken) {
	c.Header().Set(HeaderSyncToken, token.String())
}

// Conflict represents a sync conflict.
type Conflict[T any] struct {
	ID         string    `json:"id"`
	ClientData T         `json:"client_data"`
	ServerData T         `json:"server_data"`
	ClientTime time.Time `json:"client_time"`
	ServerTime time.Time `json:"server_time"`
}

// ConflictResolution is the strategy for resolving conflicts.
type ConflictResolution string

const (
	// ResolutionServerWins means server data takes precedence.
	ResolutionServerWins ConflictResolution = "server_wins"

	// ResolutionClientWins means client data takes precedence.
	ResolutionClientWins ConflictResolution = "client_wins"

	// ResolutionLatestWins means most recent timestamp wins.
	ResolutionLatestWins ConflictResolution = "latest_wins"

	// ResolutionManual means conflicts must be resolved manually.
	ResolutionManual ConflictResolution = "manual"
)

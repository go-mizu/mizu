package mobile

import (
	"encoding/base64"
	"strconv"
	"time"

	"github.com/go-mizu/mizu"
)

// SyncState represents offline sync state.
type SyncState struct {
	LastSync  time.Time `json:"last_sync"`
	SyncToken string    `json:"sync_token"`
	HasMore   bool      `json:"has_more"`
}

// SyncResponse wraps data with sync metadata.
type SyncResponse[T any] struct {
	Data      T         `json:"data"`
	SyncState SyncState `json:"sync_state"`
	Deleted   []string  `json:"deleted,omitempty"`
}

// NewSyncResponse creates a sync response.
func NewSyncResponse[T any](data T, syncTime time.Time, deleted []string, hasMore bool) SyncResponse[T] {
	return SyncResponse[T]{
		Data:    data,
		Deleted: deleted,
		SyncState: SyncState{
			LastSync:  syncTime,
			SyncToken: NewSyncToken(syncTime),
			HasMore:   hasMore,
		},
	}
}

// ParseSyncToken extracts sync token from header or query.
func ParseSyncToken(c *mizu.Ctx) string {
	if token := c.Request().Header.Get(HeaderSyncToken); token != "" {
		return token
	}
	return c.Query("sync_token")
}

// NewSyncToken generates a sync token from timestamp.
// The token encodes the timestamp in base64.
func NewSyncToken(t time.Time) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(t.UnixNano(), 36)),
	)
}

// ParseSyncTokenTime decodes a sync token to timestamp.
func ParseSyncTokenTime(token string) (time.Time, error) {
	if token == "" {
		return time.Time{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, err
	}
	ns, err := strconv.ParseInt(string(b), 36, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, ns), nil
}

// LastModified represents a resource with last modified timestamp.
type LastModified struct {
	ID         string    `json:"id"`
	ModifiedAt time.Time `json:"modified_at"`
}

// SyncDelta represents changes since last sync.
type SyncDelta[T any] struct {
	Created  []T      `json:"created,omitempty"`
	Updated  []T      `json:"updated,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
	SyncTime time.Time
}

// NewSyncDelta creates an empty sync delta.
func NewSyncDelta[T any]() *SyncDelta[T] {
	return &SyncDelta[T]{
		SyncTime: time.Now(),
	}
}

// AddCreated adds a created item.
func (d *SyncDelta[T]) AddCreated(item T) {
	d.Created = append(d.Created, item)
}

// AddUpdated adds an updated item.
func (d *SyncDelta[T]) AddUpdated(item T) {
	d.Updated = append(d.Updated, item)
}

// AddDeleted adds a deleted item ID.
func (d *SyncDelta[T]) AddDeleted(id string) {
	d.Deleted = append(d.Deleted, id)
}

// IsEmpty returns true if no changes.
func (d *SyncDelta[T]) IsEmpty() bool {
	return len(d.Created) == 0 && len(d.Updated) == 0 && len(d.Deleted) == 0
}

// Total returns total number of changes.
func (d *SyncDelta[T]) Total() int {
	return len(d.Created) + len(d.Updated) + len(d.Deleted)
}

// ToSyncResponse converts delta to sync response.
func (d *SyncDelta[T]) ToSyncResponse(hasMore bool) SyncResponse[SyncDelta[T]] {
	return NewSyncResponse(*d, d.SyncTime, nil, hasMore)
}

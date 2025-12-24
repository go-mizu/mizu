package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/members"
	"github.com/go-mizu/blueprints/chat/feature/messages"
	"github.com/go-mizu/blueprints/chat/feature/roles"
	"github.com/go-mizu/blueprints/chat/feature/servers"
)

// setupTestDB creates an in-memory DuckDB for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	store, err := New(db)
	if err != nil {
		db.Close()
		t.Fatalf("create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("ensure schema: %v", err)
	}

	return db
}

// createTestUser creates a test user and returns it.
func createTestUser(t *testing.T, store *UsersStore, username string) *accounts.User {
	t.Helper()
	now := time.Now()
	user := &accounts.User{
		ID:            "user-" + username,
		Username:      username,
		Discriminator: "0001",
		DisplayName:   username + " Display",
		Email:         username + "@example.com",
		Status:        accounts.StatusOnline,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.Insert(context.Background(), user, "hashedpassword123"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return user
}

// createTestServer creates a test server and returns it.
func createTestServer(t *testing.T, store *ServersStore, ownerID, name string) *servers.Server {
	t.Helper()
	now := time.Now()
	srv := &servers.Server{
		ID:          "server-" + name,
		Name:        name,
		Description: name + " description",
		OwnerID:     ownerID,
		IsPublic:    true,
		InviteCode:  "invite-" + name,
		MemberCount: 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Insert(context.Background(), srv); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	return srv
}

// createTestChannel creates a test channel and returns it.
func createTestChannel(t *testing.T, store *ChannelsStore, serverID, name string) *channels.Channel {
	t.Helper()
	now := time.Now()
	ch := &channels.Channel{
		ID:        "channel-" + name,
		ServerID:  serverID,
		Type:      channels.TypeText,
		Name:      name,
		Topic:     name + " topic",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Insert(context.Background(), ch); err != nil {
		t.Fatalf("insert channel: %v", err)
	}
	return ch
}

// createTestMessage creates a test message and returns it.
func createTestMessage(t *testing.T, store *MessagesStore, channelID, authorID, content string) *messages.Message {
	t.Helper()
	now := time.Now()
	msg := &messages.Message{
		ID:        "msg-" + content[:min(10, len(content))],
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
		Type:      messages.TypeDefault,
		CreatedAt: now,
	}
	if err := store.Insert(context.Background(), msg); err != nil {
		t.Fatalf("insert message: %v", err)
	}
	return msg
}

// createTestMember creates a test member and returns it.
func createTestMember(t *testing.T, store *MembersStore, serverID, userID string) *members.Member {
	t.Helper()
	m := &members.Member{
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
	if err := store.Insert(context.Background(), m); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	return m
}

// createTestRole creates a test role and returns it.
func createTestRole(t *testing.T, store *RolesStore, serverID, name string) *roles.Role {
	t.Helper()
	r := &roles.Role{
		ID:          "role-" + name,
		ServerID:    serverID,
		Name:        name,
		Color:       0xFF0000,
		Position:    1,
		Permissions: roles.PermissionViewChannel | roles.PermissionSendMessages,
		CreatedAt:   time.Now(),
	}
	if err := store.Insert(context.Background(), r); err != nil {
		t.Fatalf("insert role: %v", err)
	}
	return r
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

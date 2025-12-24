package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/messages"
)

func TestMessagesStore_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")

	msg := &messages.Message{
		ID:        "msg1",
		ChannelID: ch.ID,
		AuthorID:  owner.ID,
		Content:   "Hello, world!",
		Type:      messages.TypeDefault,
		CreatedAt: time.Now(),
	}

	err := store.Insert(ctx, msg)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	got, err := store.GetByID(ctx, "msg1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Content != msg.Content {
		t.Errorf("Content = %v, want %v", got.Content, msg.Content)
	}
}

func TestMessagesStore_InsertWithMentions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	user2 := createTestUser(t, usersStore, "user2")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")

	msg := &messages.Message{
		ID:        "msg-mention",
		ChannelID: ch.ID,
		AuthorID:  owner.ID,
		Content:   "Hey @user2!",
		Mentions:  []string{user2.ID},
		Type:      messages.TypeDefault,
		CreatedAt: time.Now(),
	}

	err := store.Insert(ctx, msg)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	got, _ := store.GetByID(ctx, msg.ID)
	if len(got.Mentions) != 1 {
		t.Errorf("len(Mentions) = %d, want 1", len(got.Mentions))
	}
}

func TestMessagesStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "Test message")

	got, err := store.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != msg.ID {
		t.Errorf("ID = %v, want %v", got.ID, msg.ID)
	}

	// Non-existent
	_, err = store.GetByID(ctx, "nonexistent")
	if err != messages.ErrNotFound {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestMessagesStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "Original message")

	newContent := "Edited message"
	err := store.Update(ctx, msg.ID, &messages.UpdateIn{
		Content: &newContent,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.GetByID(ctx, msg.ID)
	if got.Content != newContent {
		t.Errorf("Content = %v, want %v", got.Content, newContent)
	}
	if !got.IsEdited {
		t.Error("IsEdited should be true")
	}
}

func TestMessagesStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "To be deleted")

	err := store.Delete(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.GetByID(ctx, msg.ID)
	if err != messages.ErrNotFound {
		t.Errorf("GetByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestMessagesStore_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")

	// Create multiple messages
	for i := 0; i < 5; i++ {
		msg := &messages.Message{
			ID:        "msg-" + string(rune('A'+i)),
			ChannelID: ch.ID,
			AuthorID:  owner.ID,
			Content:   "Message " + string(rune('A'+i)),
			Type:      messages.TypeDefault,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		store.Insert(ctx, msg)
	}

	// List all
	msgs, err := store.List(ctx, ch.ID, messages.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(msgs) != 5 {
		t.Errorf("len(msgs) = %d, want 5", len(msgs))
	}

	// List with limit
	msgs, _ = store.List(ctx, ch.ID, messages.ListOpts{Limit: 2})
	if len(msgs) != 2 {
		t.Errorf("len(msgs) with limit = %d, want 2", len(msgs))
	}
}

func TestMessagesStore_ListWithPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")

	// Create messages with sequential IDs
	for i := 0; i < 10; i++ {
		msg := &messages.Message{
			ID:        "msg-" + string(rune('A'+i)),
			ChannelID: ch.ID,
			AuthorID:  owner.ID,
			Content:   "Message " + string(rune('A'+i)),
			Type:      messages.TypeDefault,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		store.Insert(ctx, msg)
	}

	// Test 'before' pagination
	msgs, err := store.List(ctx, ch.ID, messages.ListOpts{
		Before: "msg-E",
		Limit:  3,
	})
	if err != nil {
		t.Fatalf("List() with before error = %v", err)
	}

	// Should get messages before E (D, C, B, A in reverse order)
	if len(msgs) > 3 {
		t.Errorf("len(msgs) = %d, want <= 3", len(msgs))
	}

	// Test 'after' pagination
	msgs, err = store.List(ctx, ch.ID, messages.ListOpts{
		After: "msg-E",
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("List() with after error = %v", err)
	}

	if len(msgs) > 3 {
		t.Errorf("len(msgs) = %d, want <= 3", len(msgs))
	}
}

func TestMessagesStore_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")

	createTestMessage(t, store, ch.ID, owner.ID, "Hello world")
	createTestMessage(t, store, ch.ID, owner.ID, "Goodbye world")
	createTestMessage(t, store, ch.ID, owner.ID, "Something else")

	// Search for 'world'
	msgs, err := store.Search(ctx, messages.SearchOpts{
		Query: "world",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(msgs) != 2 {
		t.Errorf("len(msgs) = %d, want 2", len(msgs))
	}

	// Search with channel filter
	msgs, err = store.Search(ctx, messages.SearchOpts{
		ChannelID: ch.ID,
		Query:     "world",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Search() with channel error = %v", err)
	}

	if len(msgs) != 2 {
		t.Errorf("len(msgs) = %d, want 2", len(msgs))
	}
}

func TestMessagesStore_Pin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "Important message")

	// Pin message
	err := store.Pin(ctx, ch.ID, msg.ID, owner.ID)
	if err != nil {
		t.Fatalf("Pin() error = %v", err)
	}

	got, _ := store.GetByID(ctx, msg.ID)
	if !got.IsPinned {
		t.Error("IsPinned should be true")
	}

	// List pinned
	pinned, err := store.ListPinned(ctx, ch.ID)
	if err != nil {
		t.Fatalf("ListPinned() error = %v", err)
	}

	if len(pinned) != 1 {
		t.Errorf("len(pinned) = %d, want 1", len(pinned))
	}

	// Unpin
	err = store.Unpin(ctx, ch.ID, msg.ID)
	if err != nil {
		t.Fatalf("Unpin() error = %v", err)
	}

	got, _ = store.GetByID(ctx, msg.ID)
	if got.IsPinned {
		t.Error("IsPinned should be false after unpin")
	}
}

func TestMessagesStore_Reactions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	user2 := createTestUser(t, usersStore, "user2")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "React to this!")

	// Add reactions
	err := store.AddReaction(ctx, msg.ID, owner.ID, "👍")
	if err != nil {
		t.Fatalf("AddReaction() error = %v", err)
	}

	err = store.AddReaction(ctx, msg.ID, user2.ID, "👍")
	if err != nil {
		t.Fatalf("AddReaction() error = %v", err)
	}

	err = store.AddReaction(ctx, msg.ID, owner.ID, "❤️")
	if err != nil {
		t.Fatalf("AddReaction() error = %v", err)
	}

	// Get message with reactions
	got, _ := store.GetByID(ctx, msg.ID)
	if len(got.Reactions) != 2 {
		t.Errorf("len(Reactions) = %d, want 2", len(got.Reactions))
	}

	// Get reaction users
	users, err := store.GetReactionUsers(ctx, msg.ID, "👍", 10)
	if err != nil {
		t.Fatalf("GetReactionUsers() error = %v", err)
	}

	if len(users) != 2 {
		t.Errorf("len(users) = %d, want 2", len(users))
	}

	// Remove reaction
	err = store.RemoveReaction(ctx, msg.ID, owner.ID, "👍")
	if err != nil {
		t.Fatalf("RemoveReaction() error = %v", err)
	}

	users, _ = store.GetReactionUsers(ctx, msg.ID, "👍", 10)
	if len(users) != 1 {
		t.Errorf("len(users) after remove = %d, want 1", len(users))
	}
}

func TestMessagesStore_Attachments(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "Check this file")

	// Add attachment
	att := &messages.Attachment{
		ID:          "att1",
		MessageID:   msg.ID,
		Filename:    "image.png",
		ContentType: "image/png",
		Size:        1024,
		URL:         "https://example.com/image.png",
		CreatedAt:   time.Now(),
	}

	err := store.InsertAttachment(ctx, att)
	if err != nil {
		t.Fatalf("InsertAttachment() error = %v", err)
	}

	got, _ := store.GetByID(ctx, msg.ID)
	if len(got.Attachments) != 1 {
		t.Errorf("len(Attachments) = %d, want 1", len(got.Attachments))
	}

	if got.Attachments[0].Filename != "image.png" {
		t.Errorf("Filename = %v, want image.png", got.Attachments[0].Filename)
	}
}

func TestMessagesStore_Embeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewMessagesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	msg := createTestMessage(t, store, ch.ID, owner.ID, "Check this link")

	// Add embed
	embed := &messages.Embed{
		ID:          "embed1",
		Type:        "link",
		Title:       "Example Site",
		Description: "An example website",
		URL:         "https://example.com",
		Color:       0x5865F2,
	}

	err := store.InsertEmbed(ctx, msg.ID, embed)
	if err != nil {
		t.Fatalf("InsertEmbed() error = %v", err)
	}

	// Verify embed was inserted (we'd need to extend GetByID to load embeds)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM embeds WHERE message_id = ?", msg.ID).Scan(&count)
	if count != 1 {
		t.Errorf("embed count = %d, want 1", count)
	}
}

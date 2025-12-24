package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/channels"
)

func TestChannelsStore_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	ch := &channels.Channel{
		ID:        "ch1",
		ServerID:  srv.ID,
		Type:      channels.TypeText,
		Name:      "general",
		Topic:     "General discussion",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.Insert(ctx, ch)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	got, err := store.GetByID(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != ch.Name {
		t.Errorf("Name = %v, want %v", got.Name, ch.Name)
	}
}

func TestChannelsStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, store, srv.ID, "general")

	got, err := store.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != ch.ID {
		t.Errorf("ID = %v, want %v", got.ID, ch.ID)
	}

	// Non-existent
	_, err = store.GetByID(ctx, "nonexistent")
	if err != channels.ErrNotFound {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestChannelsStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, store, srv.ID, "general")

	newName := "updated-channel"
	newTopic := "Updated topic"
	err := store.Update(ctx, ch.ID, &channels.UpdateIn{
		Name:  &newName,
		Topic: &newTopic,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.GetByID(ctx, ch.ID)
	if got.Name != newName {
		t.Errorf("Name = %v, want %v", got.Name, newName)
	}
	if got.Topic != newTopic {
		t.Errorf("Topic = %v, want %v", got.Topic, newTopic)
	}
}

func TestChannelsStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, store, srv.ID, "general")

	err := store.Delete(ctx, ch.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.GetByID(ctx, ch.ID)
	if err != channels.ErrNotFound {
		t.Errorf("GetByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestChannelsStore_ListByServer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	createTestChannel(t, store, srv.ID, "general")
	createTestChannel(t, store, srv.ID, "random")
	createTestChannel(t, store, srv.ID, "announcements")

	chs, err := store.ListByServer(ctx, srv.ID)
	if err != nil {
		t.Fatalf("ListByServer() error = %v", err)
	}

	if len(chs) != 3 {
		t.Errorf("len(chs) = %d, want 3", len(chs))
	}
}

func TestChannelsStore_DMChannel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	user1 := createTestUser(t, usersStore, "user1")
	user2 := createTestUser(t, usersStore, "user2")

	// Create DM channel
	dm := &channels.Channel{
		ID:        "dm-1-2",
		Type:      channels.TypeDM,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Insert(ctx, dm)

	// Add recipients
	store.AddRecipient(ctx, dm.ID, user1.ID)
	store.AddRecipient(ctx, dm.ID, user2.ID)

	// Get DM channel
	got, err := store.GetDMChannel(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("GetDMChannel() error = %v", err)
	}

	if got.ID != dm.ID {
		t.Errorf("ID = %v, want %v", got.ID, dm.ID)
	}

	// List DMs for user
	dms, err := store.ListDMsByUser(ctx, user1.ID)
	if err != nil {
		t.Fatalf("ListDMsByUser() error = %v", err)
	}

	if len(dms) != 1 {
		t.Errorf("len(dms) = %d, want 1", len(dms))
	}
}

func TestChannelsStore_Recipients(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	user1 := createTestUser(t, usersStore, "user1")
	user2 := createTestUser(t, usersStore, "user2")
	user3 := createTestUser(t, usersStore, "user3")

	// Create group DM
	groupDM := &channels.Channel{
		ID:        "group-dm",
		Type:      channels.TypeGroupDM,
		OwnerID:   user1.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Insert(ctx, groupDM)

	// Add recipients
	store.AddRecipient(ctx, groupDM.ID, user1.ID)
	store.AddRecipient(ctx, groupDM.ID, user2.ID)
	store.AddRecipient(ctx, groupDM.ID, user3.ID)

	// Get recipients
	recipients, err := store.GetRecipients(ctx, groupDM.ID)
	if err != nil {
		t.Fatalf("GetRecipients() error = %v", err)
	}

	if len(recipients) != 3 {
		t.Errorf("len(recipients) = %d, want 3", len(recipients))
	}

	// Remove recipient
	store.RemoveRecipient(ctx, groupDM.ID, user3.ID)

	recipients, _ = store.GetRecipients(ctx, groupDM.ID)
	if len(recipients) != 2 {
		t.Errorf("len(recipients) after remove = %d, want 2", len(recipients))
	}
}

func TestChannelsStore_UpdateLastMessage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, store, srv.ID, "general")

	msgID := "msg123"
	msgTime := time.Now()

	err := store.UpdateLastMessage(ctx, ch.ID, msgID, msgTime)
	if err != nil {
		t.Fatalf("UpdateLastMessage() error = %v", err)
	}

	got, _ := store.GetByID(ctx, ch.ID)
	if got.LastMessageID != msgID {
		t.Errorf("LastMessageID = %v, want %v", got.LastMessageID, msgID)
	}
	if got.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", got.MessageCount)
	}
}

func TestChannelsStore_Categories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	// Create category
	cat := &channels.Category{
		ID:        "cat1",
		ServerID:  srv.ID,
		Name:      "Text Channels",
		Position:  0,
		CreatedAt: time.Now(),
	}

	err := store.InsertCategory(ctx, cat)
	if err != nil {
		t.Fatalf("InsertCategory() error = %v", err)
	}

	// Get category
	gotCat, err := store.GetCategory(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetCategory() error = %v", err)
	}

	if gotCat.Name != cat.Name {
		t.Errorf("Name = %v, want %v", gotCat.Name, cat.Name)
	}

	// List categories
	cats, err := store.ListCategories(ctx, srv.ID)
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}

	if len(cats) != 1 {
		t.Errorf("len(cats) = %d, want 1", len(cats))
	}

	// Delete category
	err = store.DeleteCategory(ctx, cat.ID)
	if err != nil {
		t.Fatalf("DeleteCategory() error = %v", err)
	}

	_, err = store.GetCategory(ctx, cat.ID)
	if err != channels.ErrNotFound {
		t.Errorf("GetCategory() after delete error = %v, want ErrNotFound", err)
	}
}

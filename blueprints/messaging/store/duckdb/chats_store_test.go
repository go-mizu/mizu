package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
)

func testChatsStore(t *testing.T) (*ChatsStore, *UsersStore) {
	t.Helper()
	store := testStore(t)
	return NewChatsStore(store.DB()), NewUsersStore(store.DB())
}

func createTestChat(t *testing.T, cs *ChatsStore, chatType chats.ChatType, suffix string, ownerID string) *chats.Chat {
	t.Helper()
	now := time.Now()
	c := &chats.Chat{
		ID:          "chat_" + suffix,
		Type:        chatType,
		Name:        "Test Chat " + suffix,
		Description: "A test chat",
		IconURL:     "https://example.com/icon.png",
		OwnerID:     ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := cs.Insert(context.Background(), c); err != nil {
		t.Fatalf("failed to create test chat: %v", err)
	}
	return c
}

func TestNewChatsStore(t *testing.T) {
	store := testStore(t)
	cs := NewChatsStore(store.DB())

	if cs == nil {
		t.Fatal("NewChatsStore() returned nil")
	}
	if cs.db == nil {
		t.Fatal("ChatsStore.db is nil")
	}
}

func TestChatsStore_Insert(t *testing.T) {
	cs, _ := testChatsStore(t)
	ctx := context.Background()

	t.Run("direct chat", func(t *testing.T) {
		now := time.Now()
		c := &chats.Chat{
			ID:        "chat_insert_direct",
			Type:      chats.TypeDirect,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := cs.Insert(ctx, c)
		if err != nil {
			t.Fatalf("Insert() returned error: %v", err)
		}

		retrieved, err := cs.GetByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("failed to retrieve chat: %v", err)
		}
		if retrieved.Type != chats.TypeDirect {
			t.Errorf("Type = %v, want %v", retrieved.Type, chats.TypeDirect)
		}
	})

	t.Run("group chat", func(t *testing.T) {
		now := time.Now()
		c := &chats.Chat{
			ID:          "chat_insert_group",
			Type:        chats.TypeGroup,
			Name:        "Test Group",
			Description: "A test group chat",
			IconURL:     "https://example.com/group.png",
			OwnerID:     "owner_123",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := cs.Insert(ctx, c)
		if err != nil {
			t.Fatalf("Insert() returned error: %v", err)
		}

		retrieved, err := cs.GetByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("failed to retrieve chat: %v", err)
		}
		if retrieved.Name != c.Name {
			t.Errorf("Name = %v, want %v", retrieved.Name, c.Name)
		}
		if retrieved.Description != c.Description {
			t.Errorf("Description = %v, want %v", retrieved.Description, c.Description)
		}
	})
}

func TestChatsStore_GetByID(t *testing.T) {
	cs, _ := testChatsStore(t)
	ctx := context.Background()

	t.Run("existing chat", func(t *testing.T) {
		c := createTestChat(t, cs, chats.TypeGroup, "getbyid", "owner_1")
		retrieved, err := cs.GetByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("GetByID() returned error: %v", err)
		}
		if retrieved.ID != c.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, c.ID)
		}
	})

	t.Run("non-existing chat", func(t *testing.T) {
		_, err := cs.GetByID(ctx, "nonexistent")
		if err != chats.ErrNotFound {
			t.Errorf("GetByID() error = %v, want %v", err, chats.ErrNotFound)
		}
	})
}

func TestChatsStore_GetByIDForUser(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	// Create user and chat
	u := createTestUser(t, us, "chatuser")
	c := createTestChat(t, cs, chats.TypeGroup, "getbyidforuser", u.ID)

	// Add user as participant
	p := &chats.Participant{
		ChatID:            c.ID,
		UserID:            u.ID,
		Role:              "member",
		JoinedAt:          time.Now(),
		IsMuted:           false,
		UnreadCount:       5,
		NotificationLevel: "all",
	}
	cs.InsertParticipant(ctx, p)

	t.Run("user is participant", func(t *testing.T) {
		retrieved, err := cs.GetByIDForUser(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("GetByIDForUser() returned error: %v", err)
		}
		if retrieved.ID != c.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, c.ID)
		}
		if retrieved.UnreadCount != 5 {
			t.Errorf("UnreadCount = %v, want 5", retrieved.UnreadCount)
		}
	})

	t.Run("user not participant", func(t *testing.T) {
		_, err := cs.GetByIDForUser(ctx, c.ID, "other_user")
		if err != chats.ErrNotFound {
			t.Errorf("GetByIDForUser() error = %v, want %v", err, chats.ErrNotFound)
		}
	})
}

func TestChatsStore_GetDirectChat(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	u1 := createTestUser(t, us, "direct1")
	u2 := createTestUser(t, us, "direct2")
	u3 := createTestUser(t, us, "direct3")

	// Create a direct chat between u1 and u2
	now := time.Now()
	c := &chats.Chat{
		ID:        "chat_direct_test",
		Type:      chats.TypeDirect,
		CreatedAt: now,
		UpdatedAt: now,
	}
	cs.Insert(ctx, c)
	cs.InsertParticipant(ctx, &chats.Participant{ChatID: c.ID, UserID: u1.ID, Role: "member", JoinedAt: now})
	cs.InsertParticipant(ctx, &chats.Participant{ChatID: c.ID, UserID: u2.ID, Role: "member", JoinedAt: now})

	t.Run("existing direct chat", func(t *testing.T) {
		retrieved, err := cs.GetDirectChat(ctx, u1.ID, u2.ID)
		if err != nil {
			t.Fatalf("GetDirectChat() returned error: %v", err)
		}
		if retrieved.ID != c.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, c.ID)
		}
	})

	t.Run("reverse order", func(t *testing.T) {
		retrieved, err := cs.GetDirectChat(ctx, u2.ID, u1.ID)
		if err != nil {
			t.Fatalf("GetDirectChat() returned error: %v", err)
		}
		if retrieved.ID != c.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, c.ID)
		}
	})

	t.Run("no direct chat exists", func(t *testing.T) {
		_, err := cs.GetDirectChat(ctx, u1.ID, u3.ID)
		if err != chats.ErrNotFound {
			t.Errorf("GetDirectChat() error = %v, want %v", err, chats.ErrNotFound)
		}
	})
}

func TestChatsStore_List(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	u := createTestUser(t, us, "listchats")
	now := time.Now()

	// Create multiple chats for the user
	for i := 0; i < 5; i++ {
		c := createTestChat(t, cs, chats.TypeGroup, "list"+string(rune('a'+i)), u.ID)
		cs.InsertParticipant(ctx, &chats.Participant{
			ChatID:   c.ID,
			UserID:   u.ID,
			Role:     "member",
			JoinedAt: now,
		})
	}

	t.Run("list user chats", func(t *testing.T) {
		chatList, err := cs.List(ctx, u.ID, chats.ListOpts{Limit: 10})
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}
		if len(chatList) < 5 {
			t.Errorf("len(chatList) = %v, want >= 5", len(chatList))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		chatList, err := cs.List(ctx, u.ID, chats.ListOpts{Limit: 3})
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}
		if len(chatList) != 3 {
			t.Errorf("len(chatList) = %v, want 3", len(chatList))
		}
	})

	t.Run("with offset", func(t *testing.T) {
		chatList, err := cs.List(ctx, u.ID, chats.ListOpts{Limit: 10, Offset: 2})
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}
		if len(chatList) < 3 {
			t.Errorf("len(chatList) = %v, want >= 3", len(chatList))
		}
	})

	t.Run("exclude archived", func(t *testing.T) {
		// Archive one chat
		c := createTestChat(t, cs, chats.TypeGroup, "archived", u.ID)
		cs.InsertParticipant(ctx, &chats.Participant{ChatID: c.ID, UserID: u.ID, Role: "member", JoinedAt: now})
		cs.Archive(ctx, c.ID, u.ID)

		chatList, err := cs.List(ctx, u.ID, chats.ListOpts{Limit: 100, IncludeArchived: false})
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}

		// Check archived chat is not in list
		for _, chat := range chatList {
			if chat.ID == c.ID {
				t.Error("archived chat should not be in list")
			}
		}
	})

	t.Run("include archived", func(t *testing.T) {
		chatList, err := cs.List(ctx, u.ID, chats.ListOpts{Limit: 100, IncludeArchived: true})
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}
		if len(chatList) < 6 {
			t.Errorf("len(chatList) = %v, want >= 6", len(chatList))
		}
	})
}

func TestChatsStore_Update(t *testing.T) {
	cs, _ := testChatsStore(t)
	ctx := context.Background()

	t.Run("update name", func(t *testing.T) {
		c := createTestChat(t, cs, chats.TypeGroup, "updatename", "owner")
		newName := "Updated Name"
		err := cs.Update(ctx, c.ID, &chats.UpdateIn{Name: &newName})
		if err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}

		retrieved, _ := cs.GetByID(ctx, c.ID)
		if retrieved.Name != newName {
			t.Errorf("Name = %v, want %v", retrieved.Name, newName)
		}
	})

	t.Run("update multiple fields", func(t *testing.T) {
		c := createTestChat(t, cs, chats.TypeGroup, "updatemulti", "owner")
		newName := "New Name"
		newDesc := "New Description"
		newIcon := "https://example.com/new-icon.png"

		err := cs.Update(ctx, c.ID, &chats.UpdateIn{
			Name:        &newName,
			Description: &newDesc,
			IconURL:     &newIcon,
		})
		if err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}

		retrieved, _ := cs.GetByID(ctx, c.ID)
		if retrieved.Name != newName {
			t.Errorf("Name = %v, want %v", retrieved.Name, newName)
		}
		if retrieved.Description != newDesc {
			t.Errorf("Description = %v, want %v", retrieved.Description, newDesc)
		}
		if retrieved.IconURL != newIcon {
			t.Errorf("IconURL = %v, want %v", retrieved.IconURL, newIcon)
		}
	})

	t.Run("empty update", func(t *testing.T) {
		c := createTestChat(t, cs, chats.TypeGroup, "updateempty", "owner")
		err := cs.Update(ctx, c.ID, &chats.UpdateIn{})
		if err != nil {
			t.Fatalf("Update() with no changes returned error: %v", err)
		}
	})
}

func TestChatsStore_Delete(t *testing.T) {
	cs, _ := testChatsStore(t)
	ctx := context.Background()

	t.Run("existing chat", func(t *testing.T) {
		c := createTestChat(t, cs, chats.TypeGroup, "delete", "owner")
		err := cs.Delete(ctx, c.ID)
		if err != nil {
			t.Fatalf("Delete() returned error: %v", err)
		}

		_, err = cs.GetByID(ctx, c.ID)
		if err != chats.ErrNotFound {
			t.Errorf("GetByID() after delete error = %v, want %v", err, chats.ErrNotFound)
		}
	})

	t.Run("non-existing chat", func(t *testing.T) {
		err := cs.Delete(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Delete() non-existing should not error: %v", err)
		}
	})
}

func TestChatsStore_Participants(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "participants", "owner")
	u1 := createTestUser(t, us, "participant1")
	u2 := createTestUser(t, us, "participant2")
	now := time.Now()

	t.Run("insert participant", func(t *testing.T) {
		p := &chats.Participant{
			ChatID:            c.ID,
			UserID:            u1.ID,
			Role:              "member",
			JoinedAt:          now,
			NotificationLevel: "all",
		}
		err := cs.InsertParticipant(ctx, p)
		if err != nil {
			t.Fatalf("InsertParticipant() returned error: %v", err)
		}
	})

	t.Run("get participant", func(t *testing.T) {
		p, err := cs.GetParticipant(ctx, c.ID, u1.ID)
		if err != nil {
			t.Fatalf("GetParticipant() returned error: %v", err)
		}
		if p.Role != "member" {
			t.Errorf("Role = %v, want member", p.Role)
		}
	})

	t.Run("get non-existing participant", func(t *testing.T) {
		_, err := cs.GetParticipant(ctx, c.ID, "nonexistent")
		if err != chats.ErrNotFound {
			t.Errorf("GetParticipant() error = %v, want %v", err, chats.ErrNotFound)
		}
	})

	t.Run("get participants", func(t *testing.T) {
		cs.InsertParticipant(ctx, &chats.Participant{
			ChatID:            c.ID,
			UserID:            u2.ID,
			Role:              "admin",
			JoinedAt:          now,
			NotificationLevel: "all",
		})

		participants, err := cs.GetParticipants(ctx, c.ID)
		if err != nil {
			t.Fatalf("GetParticipants() returned error: %v", err)
		}
		if len(participants) != 2 {
			t.Errorf("len(participants) = %v, want 2", len(participants))
		}
	})

	t.Run("is participant", func(t *testing.T) {
		isP, err := cs.IsParticipant(ctx, c.ID, u1.ID)
		if err != nil {
			t.Fatalf("IsParticipant() returned error: %v", err)
		}
		if !isP {
			t.Error("IsParticipant() = false, want true")
		}
	})

	t.Run("is not participant", func(t *testing.T) {
		isP, err := cs.IsParticipant(ctx, c.ID, "nonexistent")
		if err != nil {
			t.Fatalf("IsParticipant() returned error: %v", err)
		}
		if isP {
			t.Error("IsParticipant() = true, want false")
		}
	})

	t.Run("update participant role", func(t *testing.T) {
		err := cs.UpdateParticipantRole(ctx, c.ID, u1.ID, "admin")
		if err != nil {
			t.Fatalf("UpdateParticipantRole() returned error: %v", err)
		}

		p, _ := cs.GetParticipant(ctx, c.ID, u1.ID)
		if p.Role != "admin" {
			t.Errorf("Role = %v, want admin", p.Role)
		}
	})

	t.Run("delete participant", func(t *testing.T) {
		err := cs.DeleteParticipant(ctx, c.ID, u1.ID)
		if err != nil {
			t.Fatalf("DeleteParticipant() returned error: %v", err)
		}

		isP, _ := cs.IsParticipant(ctx, c.ID, u1.ID)
		if isP {
			t.Error("participant should be deleted")
		}
	})
}

func TestChatsStore_MuteUnmute(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "mute", "owner")
	u := createTestUser(t, us, "muteuser")
	now := time.Now()

	cs.InsertParticipant(ctx, &chats.Participant{
		ChatID:   c.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: now,
	})

	t.Run("mute chat", func(t *testing.T) {
		until := time.Now().Add(24 * time.Hour)
		err := cs.Mute(ctx, c.ID, u.ID, &until)
		if err != nil {
			t.Fatalf("Mute() returned error: %v", err)
		}

		p, _ := cs.GetParticipant(ctx, c.ID, u.ID)
		if !p.IsMuted {
			t.Error("IsMuted = false, want true")
		}
	})

	t.Run("mute indefinitely", func(t *testing.T) {
		err := cs.Mute(ctx, c.ID, u.ID, nil)
		if err != nil {
			t.Fatalf("Mute() returned error: %v", err)
		}

		p, _ := cs.GetParticipant(ctx, c.ID, u.ID)
		if !p.IsMuted {
			t.Error("IsMuted = false, want true")
		}
	})

	t.Run("unmute chat", func(t *testing.T) {
		err := cs.Unmute(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Unmute() returned error: %v", err)
		}

		p, _ := cs.GetParticipant(ctx, c.ID, u.ID)
		if p.IsMuted {
			t.Error("IsMuted = true, want false")
		}
	})
}

func TestChatsStore_ArchiveUnarchive(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "archive", "owner")
	u := createTestUser(t, us, "archiveuser")
	now := time.Now()

	cs.InsertParticipant(ctx, &chats.Participant{
		ChatID:   c.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: now,
	})

	t.Run("archive chat", func(t *testing.T) {
		err := cs.Archive(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Archive() returned error: %v", err)
		}

		chat, _ := cs.GetByIDForUser(ctx, c.ID, u.ID)
		if !chat.IsArchived {
			t.Error("IsArchived = false, want true")
		}
	})

	t.Run("archive idempotent", func(t *testing.T) {
		err := cs.Archive(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Archive() second call returned error: %v", err)
		}
	})

	t.Run("unarchive chat", func(t *testing.T) {
		err := cs.Unarchive(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Unarchive() returned error: %v", err)
		}

		chat, _ := cs.GetByIDForUser(ctx, c.ID, u.ID)
		if chat.IsArchived {
			t.Error("IsArchived = true, want false")
		}
	})
}

func TestChatsStore_PinUnpin(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "pin", "owner")
	u := createTestUser(t, us, "pinuser")
	now := time.Now()

	cs.InsertParticipant(ctx, &chats.Participant{
		ChatID:   c.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: now,
	})

	t.Run("pin chat", func(t *testing.T) {
		err := cs.Pin(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Pin() returned error: %v", err)
		}

		chat, _ := cs.GetByIDForUser(ctx, c.ID, u.ID)
		if !chat.IsPinned {
			t.Error("IsPinned = false, want true")
		}
	})

	t.Run("pin idempotent", func(t *testing.T) {
		err := cs.Pin(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Pin() second call returned error: %v", err)
		}
	})

	t.Run("pin multiple chats", func(t *testing.T) {
		c2 := createTestChat(t, cs, chats.TypeGroup, "pin2", "owner")
		cs.InsertParticipant(ctx, &chats.Participant{ChatID: c2.ID, UserID: u.ID, Role: "member", JoinedAt: now})

		err := cs.Pin(ctx, c2.ID, u.ID)
		if err != nil {
			t.Fatalf("Pin() returned error: %v", err)
		}
	})

	t.Run("unpin chat", func(t *testing.T) {
		err := cs.Unpin(ctx, c.ID, u.ID)
		if err != nil {
			t.Fatalf("Unpin() returned error: %v", err)
		}

		chat, _ := cs.GetByIDForUser(ctx, c.ID, u.ID)
		if chat.IsPinned {
			t.Error("IsPinned = true, want false")
		}
	})
}

func TestChatsStore_MarkAsRead(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "markread", "owner")
	u := createTestUser(t, us, "markreaduser")
	now := time.Now()

	cs.InsertParticipant(ctx, &chats.Participant{
		ChatID:   c.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: now,
	})

	t.Run("mark as read", func(t *testing.T) {
		err := cs.MarkAsRead(ctx, c.ID, u.ID, "msg_123")
		if err != nil {
			t.Fatalf("MarkAsRead() returned error: %v", err)
		}

		p, _ := cs.GetParticipant(ctx, c.ID, u.ID)
		if p.LastReadMessageID != "msg_123" {
			t.Errorf("LastReadMessageID = %v, want msg_123", p.LastReadMessageID)
		}
	})
}

func TestChatsStore_UnreadCount(t *testing.T) {
	cs, us := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "unread", "owner")
	u1 := createTestUser(t, us, "unread1")
	u2 := createTestUser(t, us, "unread2")
	now := time.Now()

	cs.InsertParticipant(ctx, &chats.Participant{ChatID: c.ID, UserID: u1.ID, Role: "member", JoinedAt: now})
	cs.InsertParticipant(ctx, &chats.Participant{ChatID: c.ID, UserID: u2.ID, Role: "member", JoinedAt: now})

	t.Run("increment unread count", func(t *testing.T) {
		err := cs.IncrementUnreadCount(ctx, c.ID, u1.ID)
		if err != nil {
			t.Fatalf("IncrementUnreadCount() returned error: %v", err)
		}

		// u2's unread count should be incremented, u1's should stay at 0
		p2, _ := cs.GetParticipant(ctx, c.ID, u2.ID)
		if p2.UnreadCount != 1 {
			t.Errorf("u2 UnreadCount = %v, want 1", p2.UnreadCount)
		}

		p1, _ := cs.GetParticipant(ctx, c.ID, u1.ID)
		if p1.UnreadCount != 0 {
			t.Errorf("u1 UnreadCount = %v, want 0", p1.UnreadCount)
		}
	})

	t.Run("reset unread count", func(t *testing.T) {
		cs.IncrementUnreadCount(ctx, c.ID, u1.ID) // Increment u2's count again

		err := cs.ResetUnreadCount(ctx, c.ID, u2.ID)
		if err != nil {
			t.Fatalf("ResetUnreadCount() returned error: %v", err)
		}

		p2, _ := cs.GetParticipant(ctx, c.ID, u2.ID)
		if p2.UnreadCount != 0 {
			t.Errorf("UnreadCount = %v, want 0", p2.UnreadCount)
		}
	})
}

func TestChatsStore_MessageCount(t *testing.T) {
	cs, _ := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "msgcount", "owner")

	t.Run("increment message count", func(t *testing.T) {
		err := cs.IncrementMessageCount(ctx, c.ID)
		if err != nil {
			t.Fatalf("IncrementMessageCount() returned error: %v", err)
		}

		chat, _ := cs.GetByID(ctx, c.ID)
		if chat.MessageCount != 1 {
			t.Errorf("MessageCount = %v, want 1", chat.MessageCount)
		}
	})

	t.Run("increment multiple times", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			cs.IncrementMessageCount(ctx, c.ID)
		}

		chat, _ := cs.GetByID(ctx, c.ID)
		if chat.MessageCount != 6 {
			t.Errorf("MessageCount = %v, want 6", chat.MessageCount)
		}
	})
}

func TestChatsStore_UpdateLastMessage(t *testing.T) {
	cs, _ := testChatsStore(t)
	ctx := context.Background()

	c := createTestChat(t, cs, chats.TypeGroup, "lastmsg", "owner")

	t.Run("update last message", func(t *testing.T) {
		before := time.Now()
		err := cs.UpdateLastMessage(ctx, c.ID, "msg_last_123")
		if err != nil {
			t.Fatalf("UpdateLastMessage() returned error: %v", err)
		}
		after := time.Now()

		chat, _ := cs.GetByID(ctx, c.ID)
		if chat.LastMessageID != "msg_last_123" {
			t.Errorf("LastMessageID = %v, want msg_last_123", chat.LastMessageID)
		}
		if chat.LastMessageAt.Before(before) || chat.LastMessageAt.After(after) {
			t.Errorf("LastMessageAt = %v, want between %v and %v", chat.LastMessageAt, before, after)
		}
	})
}

// Helper function to create test user in chats tests
func createTestUserForChats(t *testing.T, us *UsersStore, suffix string) *accounts.User {
	t.Helper()
	now := time.Now()
	u := &accounts.User{
		ID:                  "user_chat_" + suffix,
		Username:            "chatuser" + suffix,
		DisplayName:         "Chat User " + suffix,
		PrivacyLastSeen:     "everyone",
		PrivacyProfilePhoto: "everyone",
		PrivacyAbout:        "everyone",
		PrivacyGroups:       "everyone",
		PrivacyReadReceipts: true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := us.Insert(context.Background(), u, "hash"); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

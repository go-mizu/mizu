package cli

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
	"github.com/go-mizu/blueprints/messaging/store/duckdb"
)

func TestRunSeed(t *testing.T) {
	t.Run("creates sample data", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize database first
		oldDataDir := dataDir
		dataDir = tmpDir
		defer func() { dataDir = oldDataDir }()

		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit() returned error: %v", err)
		}

		// Run seed
		if err := runSeed(nil, nil); err != nil {
			t.Fatalf("runSeed() returned error: %v", err)
		}

		// Verify users were created
		dbPath := filepath.Join(tmpDir, "messaging.duckdb")
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db.Close()

		usersStore := duckdb.NewUsersStore(db)
		accountsSvc := accounts.NewService(usersStore)

		// Check alice
		alice, err := accountsSvc.GetByUsername(context.Background(), "alice")
		if err != nil {
			t.Errorf("failed to get alice: %v", err)
		}
		if alice != nil && alice.DisplayName != "Alice Smith" {
			t.Errorf("expected alice display name 'Alice Smith', got '%s'", alice.DisplayName)
		}

		// Check bob
		bob, err := accountsSvc.GetByUsername(context.Background(), "bob")
		if err != nil {
			t.Errorf("failed to get bob: %v", err)
		}
		if bob != nil && bob.DisplayName != "Bob Johnson" {
			t.Errorf("expected bob display name 'Bob Johnson', got '%s'", bob.DisplayName)
		}

		// Check charlie
		charlie, err := accountsSvc.GetByUsername(context.Background(), "charlie")
		if err != nil {
			t.Errorf("failed to get charlie: %v", err)
		}
		if charlie != nil && charlie.DisplayName != "Charlie Brown" {
			t.Errorf("expected charlie display name 'Charlie Brown', got '%s'", charlie.DisplayName)
		}

		// Verify chats were created
		chatsStore := duckdb.NewChatsStore(db)
		chatsSvc := chats.NewService(chatsStore)

		if alice != nil {
			chatList, err := chatsSvc.List(context.Background(), alice.ID, chats.ListOpts{Limit: 10})
			if err != nil {
				t.Errorf("failed to list chats: %v", err)
			}
			if len(chatList) < 2 {
				t.Errorf("expected at least 2 chats for alice, got %d", len(chatList))
			}
		}

		// Verify messages were created
		if alice != nil {
			chatsStore := duckdb.NewChatsStore(db)
			chatsSvc := chats.NewService(chatsStore)

			chatList, _ := chatsSvc.List(context.Background(), alice.ID, chats.ListOpts{Limit: 10})
			if len(chatList) > 0 {
				messagesStore := duckdb.NewMessagesStore(db)
				messagesSvc := messages.NewService(messagesStore)

				msgs, err := messagesSvc.List(context.Background(), chatList[0].ID, messages.ListOpts{Limit: 10})
				if err != nil {
					t.Errorf("failed to list messages: %v", err)
				}
				if len(msgs) == 0 {
					t.Error("expected at least 1 message")
				}
			}
		}
	})

	t.Run("idempotent - skips existing users", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDataDir := dataDir
		dataDir = tmpDir
		defer func() { dataDir = oldDataDir }()

		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit() returned error: %v", err)
		}

		// Run seed twice - should not error
		if err := runSeed(nil, nil); err != nil {
			t.Fatalf("first runSeed() returned error: %v", err)
		}

		// Second seed should skip existing users
		if err := runSeed(nil, nil); err != nil {
			t.Fatalf("second runSeed() returned error: %v", err)
		}

		// Verify still only 4 users (alice, bob, charlie + mizu-agent)
		dbPath := filepath.Join(tmpDir, "messaging.duckdb")
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count users: %v", err)
		}
		if count != 4 {
			t.Errorf("expected 4 users (alice, bob, charlie + mizu-agent), got %d", count)
		}
	})
}

func TestSeedCommand(t *testing.T) {
	cmd := NewSeed()

	if cmd.Use != "seed" {
		t.Errorf("expected Use to be 'seed', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestSeedIntegration(t *testing.T) {
	// Full integration test with all services
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.duckdb")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	usersStore := duckdb.NewUsersStore(db)
	chatsStore := duckdb.NewChatsStore(db)
	messagesStore := duckdb.NewMessagesStore(db)

	accountsSvc := accounts.NewService(usersStore)
	chatsSvc := chats.NewService(chatsStore)
	messagesSvc := messages.NewService(messagesStore)

	ctx := context.Background()

	// Create users
	alice, err := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "alice",
		Email:       "alice@test.com",
		Password:    "password123",
		DisplayName: "Alice Test",
	})
	if err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}

	bob, err := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "bob",
		Email:       "bob@test.com",
		Password:    "password123",
		DisplayName: "Bob Test",
	})
	if err != nil {
		t.Fatalf("failed to create bob: %v", err)
	}

	// Create direct chat
	chat, err := chatsSvc.CreateDirect(ctx, alice.ID, &chats.CreateDirectIn{
		RecipientID: bob.ID,
	})
	if err != nil {
		t.Fatalf("failed to create chat: %v", err)
	}

	// Create messages
	msg1, err := messagesSvc.Create(ctx, alice.ID, &messages.CreateIn{
		ChatID:  chat.ID,
		Type:    messages.TypeText,
		Content: "Hello Bob!",
	})
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	msg2, err := messagesSvc.Create(ctx, bob.ID, &messages.CreateIn{
		ChatID:  chat.ID,
		Type:    messages.TypeText,
		Content: "Hi Alice!",
	})
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Verify messages
	msgList, err := messagesSvc.List(ctx, chat.ID, messages.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if len(msgList) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgList))
	}

	// Verify message content
	foundMsg1 := false
	foundMsg2 := false
	for _, m := range msgList {
		if m.ID == msg1.ID && m.Content == "Hello Bob!" {
			foundMsg1 = true
		}
		if m.ID == msg2.ID && m.Content == "Hi Alice!" {
			foundMsg2 = true
		}
	}

	if !foundMsg1 {
		t.Error("message 1 not found or content mismatch")
	}
	if !foundMsg2 {
		t.Error("message 2 not found or content mismatch")
	}
}

func TestSeedGroupChat(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.duckdb")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	usersStore := duckdb.NewUsersStore(db)
	chatsStore := duckdb.NewChatsStore(db)

	accountsSvc := accounts.NewService(usersStore)
	chatsSvc := chats.NewService(chatsStore)

	ctx := context.Background()

	// Create users
	alice, _ := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "alice",
		Email:       "alice@test.com",
		Password:    "password123",
		DisplayName: "Alice",
	})

	bob, _ := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "bob",
		Email:       "bob@test.com",
		Password:    "password123",
		DisplayName: "Bob",
	})

	charlie, _ := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "charlie",
		Email:       "charlie@test.com",
		Password:    "password123",
		DisplayName: "Charlie",
	})

	// Create group chat
	group, err := chatsSvc.CreateGroup(ctx, alice.ID, &chats.CreateGroupIn{
		Name:           "Test Group",
		Description:    "A test group",
		ParticipantIDs: []string{bob.ID, charlie.ID},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	if group.Name != "Test Group" {
		t.Errorf("expected group name 'Test Group', got '%s'", group.Name)
	}

	if group.Type != chats.TypeGroup {
		t.Errorf("expected group type %s, got %s", chats.TypeGroup, group.Type)
	}

	// Verify all users can see the chat
	for _, user := range []*accounts.User{alice, bob, charlie} {
		chatList, err := chatsSvc.List(ctx, user.ID, chats.ListOpts{Limit: 10})
		if err != nil {
			t.Errorf("failed to list chats for %s: %v", user.Username, err)
		}
		found := false
		for _, c := range chatList {
			if c.ID == group.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("user %s should see the group chat", user.Username)
		}
	}
}

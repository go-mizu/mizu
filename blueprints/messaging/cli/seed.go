package cli

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
	"github.com/go-mizu/blueprints/messaging/store/duckdb"
)

// AgentUsername is the username for the system agent user.
const AgentUsername = "mizu-agent"

// NewSeed creates the seed command.
func NewSeed() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long:  `Adds sample users, chats, and messages for testing.`,
		RunE:  runSeed,
	}
}

func runSeed(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconSeed, "Seeding Database")
	ui.Blank()

	dbPath := filepath.Join(dataDir, "messaging.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		ui.Error("Failed to open database")
		return err
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		ui.Error("Failed to create store")
		return err
	}

	if err := store.Ensure(context.Background()); err != nil {
		ui.Error("Failed to ensure schema")
		return err
	}

	usersStore := duckdb.NewUsersStore(db)
	chatsStore := duckdb.NewChatsStore(db)
	messagesStore := duckdb.NewMessagesStore(db)

	accountsSvc := accounts.NewService(usersStore)
	chatsSvc := chats.NewService(chatsStore)
	messagesSvc := messages.NewService(messagesStore)

	ctx := context.Background()

	// Create Mizu Agent (system user)
	ui.StartSpinner("Creating Mizu Agent...")
	start := time.Now()

	agent, err := EnsureAgent(ctx, accountsSvc)
	if err != nil {
		ui.StopSpinnerError("Failed to create agent")
		return err
	}

	ui.StopSpinner("Mizu Agent ready", time.Since(start))

	// Create sample users
	ui.StartSpinner("Creating sample users...")
	start = time.Now()

	alice, err := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "alice",
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice Smith",
	})
	if err != nil && err != accounts.ErrUsernameTaken {
		ui.StopSpinnerError("Failed to create alice")
		return err
	}
	if err == accounts.ErrUsernameTaken {
		alice, _ = accountsSvc.GetByUsername(ctx, "alice")
	}

	bob, err := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "bob",
		Email:       "bob@example.com",
		Password:    "password123",
		DisplayName: "Bob Johnson",
	})
	if err != nil && err != accounts.ErrUsernameTaken {
		ui.StopSpinnerError("Failed to create bob")
		return err
	}
	if err == accounts.ErrUsernameTaken {
		bob, _ = accountsSvc.GetByUsername(ctx, "bob")
	}

	charlie, err := accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    "charlie",
		Email:       "charlie@example.com",
		Password:    "password123",
		DisplayName: "Charlie Brown",
	})
	if err != nil && err != accounts.ErrUsernameTaken {
		ui.StopSpinnerError("Failed to create charlie")
		return err
	}
	if err == accounts.ErrUsernameTaken {
		charlie, _ = accountsSvc.GetByUsername(ctx, "charlie")
	}

	ui.StopSpinner("Users created", time.Since(start))

	// Setup default chats for each user (Saved Messages + Agent chat)
	ui.StartSpinner("Setting up default chats...")
	start = time.Now()

	for _, user := range []*accounts.User{alice, bob, charlie} {
		if user != nil {
			SetupDefaultChats(ctx, chatsSvc, messagesSvc, user.ID, agent.ID)
		}
	}

	ui.StopSpinner("Default chats created", time.Since(start))

	// Create sample chats between users
	if alice != nil && bob != nil {
		ui.StartSpinner("Creating sample chats...")
		start = time.Now()

		// Direct chat between Alice and Bob
		directChat, err := chatsSvc.CreateDirect(ctx, alice.ID, &chats.CreateDirectIn{
			RecipientID: bob.ID,
		})
		if err != nil {
			ui.StopSpinnerError("Failed to create direct chat")
			return err
		}

		// Create sample messages
		sampleMessages := []struct {
			senderID string
			content  string
		}{
			{alice.ID, "Hey Bob! How are you?"},
			{bob.ID, "Hi Alice! I'm doing great, thanks for asking!"},
			{alice.ID, "That's wonderful to hear!"},
			{bob.ID, "How's your project going?"},
			{alice.ID, "It's coming along nicely. Almost done!"},
		}

		for _, msg := range sampleMessages {
			messagesSvc.Create(ctx, msg.senderID, &messages.CreateIn{
				ChatID:  directChat.ID,
				Type:    messages.TypeText,
				Content: msg.content,
			})
		}

		ui.StopSpinner("Chats created", time.Since(start))

		// Create group chat if charlie exists
		if charlie != nil {
			ui.StartSpinner("Creating group chat...")
			start = time.Now()

			groupChat, err := chatsSvc.CreateGroup(ctx, alice.ID, &chats.CreateGroupIn{
				Name:           "Project Team",
				Description:    "Our project discussion group",
				ParticipantIDs: []string{bob.ID, charlie.ID},
			})
			if err != nil {
				ui.StopSpinnerError("Failed to create group chat")
				return err
			}

			groupMessages := []struct {
				senderID string
				content  string
			}{
				{alice.ID, "Welcome to the Project Team group!"},
				{bob.ID, "Thanks for adding me!"},
				{charlie.ID, "Hello everyone! Excited to be here."},
				{alice.ID, "Let's use this group for all project updates."},
			}

			for _, msg := range groupMessages {
				messagesSvc.Create(ctx, msg.senderID, &messages.CreateIn{
					ChatID:  groupChat.ID,
					Type:    messages.TypeText,
					Content: msg.content,
				})
			}

			ui.StopSpinner("Group chat created", time.Since(start))
		}
	}

	ui.Blank()
	ui.Summary([][2]string{
		{"Users", "alice, bob, charlie"},
		{"Agent", "mizu-agent"},
		{"Password", "password123"},
		{"Status", "Ready"},
	})

	ui.Blank()
	ui.Success("Database seeded successfully!")

	return nil
}

// EnsureAgent creates or retrieves the Mizu Agent system user.
func EnsureAgent(ctx context.Context, accountsSvc accounts.API) (*accounts.User, error) {
	// Try to get existing agent
	agent, err := accountsSvc.GetByUsername(ctx, AgentUsername)
	if err == nil && agent != nil {
		return agent, nil
	}

	// Create the agent user
	agent, err = accountsSvc.Create(ctx, &accounts.CreateIn{
		Username:    AgentUsername,
		Email:       "agent@mizu.dev",
		Password:    "agent-system-password-not-for-login",
		DisplayName: "Mizu Agent",
	})
	if err != nil && err != accounts.ErrUsernameTaken {
		return nil, err
	}
	if err == accounts.ErrUsernameTaken {
		return accountsSvc.GetByUsername(ctx, AgentUsername)
	}

	return agent, nil
}

// SetupDefaultChats creates the default chats for a new user:
// 1. Saved Messages (self-chat) with a welcome message
// 2. Chat with Mizu Agent with a welcome message
func SetupDefaultChats(ctx context.Context, chatsSvc chats.API, messagesSvc messages.API, userID, agentID string) {
	// Create Saved Messages (self-chat)
	savedChat, err := chatsSvc.CreateDirect(ctx, userID, &chats.CreateDirectIn{
		RecipientID: userID,
	})
	if err == nil && savedChat != nil {
		// Add a welcome message to Saved Messages
		messagesSvc.Create(ctx, userID, &messages.CreateIn{
			ChatID:  savedChat.ID,
			Type:    messages.TypeText,
			Content: "Welcome to Saved Messages! Use this space to save notes, links, and reminders to yourself.",
		})
	}

	// Create chat with Mizu Agent
	agentChat, err := chatsSvc.CreateDirect(ctx, userID, &chats.CreateDirectIn{
		RecipientID: agentID,
	})
	if err == nil && agentChat != nil {
		// Add a welcome message from the agent
		messagesSvc.Create(ctx, agentID, &messages.CreateIn{
			ChatID:  agentChat.ID,
			Type:    messages.TypeText,
			Content: "Hello! I'm Mizu Agent, your friendly assistant. I'm here to help you get started with messaging. Feel free to ask me anything!",
		})
	}
}

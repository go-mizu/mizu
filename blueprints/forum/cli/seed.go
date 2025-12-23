package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

// NewSeed creates the seed command with subcommands.
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with data",
		Long: `Populates the database with sample data or real data from external sources.

Subcommands:
  sample   - Create sample users, boards, threads, and comments
  reddit   - Import real posts from Reddit subreddits`,
	}

	cmd.AddCommand(
		NewSeedSample(),
		NewSeedReddit(),
		NewSeedHN(),
	)

	return cmd
}

// NewSeedSample creates the seed sample command.
func NewSeedSample() *cobra.Command {
	return &cobra.Command{
		Use:   "sample",
		Short: "Seed with sample data",
		Long: `Populates the database with sample users, boards, threads, and comments for testing.

This is useful for development and demonstration purposes.`,
		RunE: runSeed,
	}
}

func runSeed(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Seeding Forum Database")
	ui.Blank()

	// Open database
	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database opened", time.Since(start))

	ctx := context.Background()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Track counts
	var userCount, boardCount, threadCount, commentCount int

	// Create sample users
	ui.Header(iconUser, "Creating Users")
	users := []struct {
		username string
		email    string
		password string
		isAdmin  bool
	}{
		{"admin", "admin@example.com", "password123", true},
		{"alice", "alice@example.com", "password123", false},
		{"bob", "bob@example.com", "password123", false},
		{"charlie", "charlie@example.com", "password123", false},
	}

	userIDs := make(map[string]string)
	for _, u := range users {
		account, err := accountsSvc.Create(ctx, accounts.CreateIn{
			Username: u.username,
			Email:    u.email,
			Password: u.password,
		})
		if err != nil {
			// Try to get existing user
			existing, _ := accountsSvc.GetByUsername(ctx, u.username)
			if existing != nil {
				userIDs[u.username] = existing.ID
				ui.Warn(fmt.Sprintf("User @%s already exists", u.username))
			}
			continue
		}
		userIDs[u.username] = account.ID
		userCount++
		ui.UserRow(u.username, u.email, u.isAdmin, false)
	}

	// Create sample boards
	ui.Header(iconBoard, "Creating Boards")
	sampleBoards := []struct {
		name        string
		title       string
		description string
		owner       string
	}{
		{"programming", "Programming", "Discuss programming languages, tools, and techniques", "admin"},
		{"golang", "Go Programming", "Everything about the Go programming language", "alice"},
		{"webdev", "Web Development", "Web development discussions and resources", "bob"},
		{"random", "Random", "Off-topic discussions and fun stuff", "charlie"},
		{"news", "Tech News", "Latest technology news and announcements", "admin"},
	}

	boardIDs := make(map[string]string)
	for _, b := range sampleBoards {
		ownerID := userIDs[b.owner]
		if ownerID == "" {
			ownerID = userIDs["admin"]
		}
		board, err := boardsSvc.Create(ctx, ownerID, boards.CreateIn{
			Name:        b.name,
			Title:       b.title,
			Description: b.description,
		})
		if err != nil {
			// Try to get existing board
			existing, _ := boardsSvc.GetByName(ctx, b.name)
			if existing != nil {
				boardIDs[b.name] = existing.ID
				ui.Warn(fmt.Sprintf("Board b/%s already exists", b.name))
			}
			continue
		}
		boardIDs[b.name] = board.ID
		boardCount++
		ui.BoardRow(b.name, b.title, 0)
	}

	// Create sample threads
	ui.Header(iconThread, "Creating Threads")
	sampleThreads := []struct {
		board   string
		author  string
		title   string
		content string
	}{
		{
			board:   "golang",
			author:  "alice",
			title:   "Why I love Go's simplicity",
			content: "Go's simplicity is its greatest strength. The language is easy to learn, easy to read, and easy to maintain. What are your favorite things about Go?",
		},
		{
			board:   "golang",
			author:  "bob",
			title:   "Best practices for error handling in Go",
			content: "I've been exploring different patterns for error handling in Go. Here are some patterns I've found useful:\n\n1. Wrap errors with context\n2. Use sentinel errors for known conditions\n3. Create custom error types when needed",
		},
		{
			board:   "programming",
			author:  "charlie",
			title:   "What's your favorite programming language and why?",
			content: "I'm curious about what languages everyone prefers and the reasons behind their choices. Let's have a friendly discussion!",
		},
		{
			board:   "webdev",
			author:  "alice",
			title:   "Building a forum with Go and HTMX",
			content: "I've been experimenting with building a modern forum using Go for the backend and HTMX for interactivity. No JavaScript framework needed!",
		},
		{
			board:   "random",
			author:  "bob",
			title:   "Weekend project ideas?",
			content: "Looking for some fun weekend project ideas. What are you all working on in your spare time?",
		},
	}

	threadIDs := make(map[string]string)
	for i, t := range sampleThreads {
		boardID := boardIDs[t.board]
		authorID := userIDs[t.author]
		if boardID == "" || authorID == "" {
			continue
		}
		thread, err := threadsSvc.Create(ctx, authorID, threads.CreateIn{
			BoardID: boardID,
			Title:   t.title,
			Content: t.content,
			Type:    "text",
		})
		if err != nil {
			ui.Warn(fmt.Sprintf("Could not create thread: %v", err))
			continue
		}
		threadIDs[fmt.Sprintf("thread_%d", i)] = thread.ID
		threadCount++
		ui.ThreadRow(t.title, t.author, 0, 0)
	}

	// Create sample comments
	ui.Header(iconComment, "Creating Comments")
	for key, threadID := range threadIDs {
		commentAuthors := []string{"alice", "bob", "charlie", "admin"}
		commentTexts := []string{
			"Great post! I completely agree with your points.",
			"Interesting perspective. Have you considered the alternative approach?",
			"Thanks for sharing! This is really helpful.",
			"I have a different opinion on this. Let me explain...",
		}

		for i, text := range commentTexts {
			authorID := userIDs[commentAuthors[i%len(commentAuthors)]]
			if authorID == "" {
				continue
			}
			_, err := commentsSvc.Create(ctx, authorID, comments.CreateIn{
				ThreadID: threadID,
				Content:  text,
			})
			if err != nil {
				ui.Warn(fmt.Sprintf("Could not create comment on %s: %v", key, err))
				continue
			}
			commentCount++
		}
	}
	ui.Progress(iconCheck, fmt.Sprintf("Created %d comments", commentCount))

	// Summary
	ui.Summary([][2]string{
		{"Users", fmt.Sprintf("%d", userCount)},
		{"Boards", fmt.Sprintf("%d", boardCount)},
		{"Threads", fmt.Sprintf("%d", threadCount)},
		{"Comments", fmt.Sprintf("%d", commentCount)},
	})

	ui.Success("Database seeded successfully!")
	ui.Blank()
	ui.Hint("Next: run 'forum serve' to start the server")

	return nil
}

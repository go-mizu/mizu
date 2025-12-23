package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with sample data",
	Long:  `Populates the database with sample users, boards, threads, and comments for testing.`,
	RunE:  runSeed,
}

func runSeed(cmd *cobra.Command, args []string) error {
	fmt.Printf("Seeding database at %s...\n", dataDir)

	store, err := duckdb.Open(dataDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Create sample users
	fmt.Println("Creating sample users...")
	users := []struct {
		username string
		email    string
		password string
	}{
		{"admin", "admin@example.com", "password123"},
		{"alice", "alice@example.com", "password123"},
		{"bob", "bob@example.com", "password123"},
		{"charlie", "charlie@example.com", "password123"},
	}

	userIDs := make(map[string]string)
	for _, u := range users {
		account, err := accountsSvc.Create(ctx, accounts.CreateIn{
			Username: u.username,
			Email:    u.email,
			Password: u.password,
		})
		if err != nil {
			fmt.Printf("  Warning: could not create user %s: %v\n", u.username, err)
			// Try to get existing user
			existing, _ := accountsSvc.GetByUsername(ctx, u.username)
			if existing != nil {
				userIDs[u.username] = existing.ID
			}
			continue
		}
		userIDs[u.username] = account.ID
		fmt.Printf("  Created user: %s\n", u.username)
	}

	// Create sample boards
	fmt.Println("Creating sample boards...")
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
			fmt.Printf("  Warning: could not create board %s: %v\n", b.name, err)
			// Try to get existing board
			existing, _ := boardsSvc.GetByName(ctx, b.name)
			if existing != nil {
				boardIDs[b.name] = existing.ID
			}
			continue
		}
		boardIDs[b.name] = board.ID
		fmt.Printf("  Created board: b/%s\n", b.name)
	}

	// Create sample threads
	fmt.Println("Creating sample threads...")
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
			fmt.Printf("  Warning: could not create thread: %v\n", err)
			continue
		}
		threadIDs[fmt.Sprintf("thread_%d", i)] = thread.ID
		title := t.title
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		fmt.Printf("  Created thread: %s\n", title)
	}

	// Create sample comments
	fmt.Println("Creating sample comments...")
	for key, threadID := range threadIDs {
		// Add a few comments to each thread
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
				fmt.Printf("  Warning: could not create comment on %s: %v\n", key, err)
				continue
			}
		}
	}
	fmt.Println("  Created sample comments")

	// Summary
	fmt.Println("\nSeeding complete!")
	fmt.Printf("  Users: %d\n", len(userIDs))
	fmt.Printf("  Boards: %d\n", len(boardIDs))
	fmt.Printf("  Threads: %d\n", len(threadIDs))
	fmt.Println("\nYou can now run: forum serve")

	return nil
}

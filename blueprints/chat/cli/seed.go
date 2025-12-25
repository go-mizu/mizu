package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/members"
	"github.com/go-mizu/blueprints/chat/feature/messages"
	"github.com/go-mizu/blueprints/chat/feature/roles"
	"github.com/go-mizu/blueprints/chat/feature/servers"
	"github.com/go-mizu/blueprints/chat/store/duckdb"
)

// NewSeed creates the seed command.
func NewSeed() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed sample data",
		Long: `Seed the Chat database with sample data.

Creates sample users, servers, channels, and messages for testing.`,
		RunE: runSeed,
	}
}

func runSeed(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconInfo, "Seeding Chat Database")
	ui.Blank()

	// Setup
	ui.StartSpinner("Opening database...")
	start := time.Now()

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		ui.StopSpinnerError("Failed to create data directory")
		return fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "chat.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		ui.StopSpinnerError("Failed to create store")
		return fmt.Errorf("create store: %w", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		ui.StopSpinnerError("Failed to ensure schema")
		return fmt.Errorf("ensure schema: %w", err)
	}

	ui.StopSpinner("Database ready", time.Since(start))

	ctx := context.Background()

	// Create stores and services
	usersStore := duckdb.NewUsersStore(db)
	serversStore := duckdb.NewServersStore(db)
	channelsStore := duckdb.NewChannelsStore(db)
	messagesStore := duckdb.NewMessagesStore(db)
	membersStore := duckdb.NewMembersStore(db)
	rolesStore := duckdb.NewRolesStore(db)

	accountsSvc := accounts.NewService(usersStore)
	serversSvc := servers.NewService(serversStore)
	channelsSvc := channels.NewService(channelsStore)
	messagesSvc := messages.NewService(messagesStore)
	membersSvc := members.NewService(membersStore)
	rolesSvc := roles.NewService(rolesStore, nil)

	// Create sample users
	ui.Blank()
	ui.Step("Creating users...")
	users := []struct {
		Username    string
		Email       string
		Password    string
		DisplayName string
	}{
		{"alice", "alice@example.com", "password123", "Alice"},
		{"bob", "bob@example.com", "password123", "Bob"},
		{"charlie", "charlie@example.com", "password123", "Charlie"},
		{"diana", "diana@example.com", "password123", "Diana"},
	}

	var createdUsers []*accounts.User
	for _, u := range users {
		user, err := accountsSvc.Create(ctx, &accounts.CreateIn{
			Username:    u.Username,
			Email:       u.Email,
			Password:    u.Password,
			DisplayName: u.DisplayName,
		})
		if err != nil {
			ui.Item("Skip", fmt.Sprintf("%s (exists)", u.Username))
			continue
		}
		createdUsers = append(createdUsers, user)
		ui.Item("User", user.Username)
	}

	if len(createdUsers) == 0 {
		ui.Blank()
		ui.Warn("Database already seeded")
		return nil
	}

	// Create sample servers
	ui.Blank()
	ui.Step("Creating servers...")
	sampleServers := []struct {
		Name        string
		Description string
		IsPublic    bool
	}{
		{"General", "A general discussion server for everyone", true},
		{"Gaming", "For gamers and game discussions", true},
		{"Tech Talk", "Technology and programming discussions", true},
	}

	for i, s := range sampleServers {
		owner := createdUsers[i%len(createdUsers)]

		srv, err := serversSvc.Create(ctx, owner.ID, &servers.CreateIn{
			Name:        s.Name,
			Description: s.Description,
			IsPublic:    s.IsPublic,
		})
		if err != nil {
			continue
		}
		ui.Item("Server", srv.Name)

		// Create default role
		rolesSvc.CreateDefaultRole(ctx, srv.ID)

		// Create channels
		general, _ := channelsSvc.Create(ctx, &channels.CreateIn{
			ServerID: srv.ID,
			Type:     channels.TypeText,
			Name:     "general",
			Topic:    "General discussion",
		})

		channelsSvc.Create(ctx, &channels.CreateIn{
			ServerID: srv.ID,
			Type:     channels.TypeText,
			Name:     "off-topic",
			Topic:    "Random conversations",
		})

		channelsSvc.Create(ctx, &channels.CreateIn{
			ServerID: srv.ID,
			Type:     channels.TypeText,
			Name:     "announcements",
			Topic:    "Server announcements",
		})

		// Set default channel
		if general != nil {
			serversSvc.SetDefaultChannel(ctx, srv.ID, general.ID)
		}

		// Add owner as member
		membersSvc.Join(ctx, srv.ID, owner.ID)

		// Add other users as members
		for _, u := range createdUsers {
			if u.ID != owner.ID {
				membersSvc.Join(ctx, srv.ID, u.ID)
				serversSvc.IncrementMemberCount(ctx, srv.ID)
			}
		}

		// Add sample messages with rich markdown content
		if general != nil {
			sampleMessages := []string{
				"Welcome to the server! 🎉",
				"Hey everyone! I've been exploring some new **Go patterns** and wanted to share.",
				`Here's a quick example of a clean HTTP handler pattern:

` + "```go" + `
func HandleUsers(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        users, err := db.Query("SELECT id, name FROM users")
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        defer users.Close()

        json.NewEncoder(w).Encode(users)
    }
}
` + "```" + `

This keeps your handlers clean and testable!`,
				`Great question! Here's what I recommend:

## Best Practices for API Design

1. **Use consistent naming conventions** - stick to snake_case or camelCase
2. **Version your APIs** - use ` + "`/api/v1/`" + ` prefixes
3. **Return proper status codes** - don't just use 200 for everything

### Example Response Structure

` + "```json" + `
{
  "data": {
    "id": "user_123",
    "name": "Alice",
    "email": "alice@example.com"
  },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
` + "```" + `

> Pro tip: Always include pagination metadata for list endpoints!`,
				`I've been working on a comparison of popular Go web frameworks:

| Framework | Stars | Performance | Ease of Use |
|-----------|-------|-------------|-------------|
| Gin       | 75k   | Excellent   | Good        |
| Echo      | 28k   | Excellent   | Excellent   |
| Fiber     | 30k   | Best        | Good        |
| Chi       | 16k   | Very Good   | Excellent   |
| **Mizu**  | 1k    | Excellent   | Amazing!    |

Each has its strengths. What are you all using?`,
				`Here's a Python script I use for quick data processing:

` + "```python" + `
import pandas as pd
from pathlib import Path

def process_logs(log_dir: str) -> pd.DataFrame:
    """Process all log files in a directory."""
    logs = []

    for file in Path(log_dir).glob("*.log"):
        with open(file) as f:
            for line in f:
                if "ERROR" in line:
                    logs.append({
                        "file": file.name,
                        "line": line.strip()
                    })

    return pd.DataFrame(logs)

# Usage
df = process_logs("/var/log/myapp")
print(f"Found {len(df)} errors")
` + "```" + `

Super handy for debugging production issues!`,
				`Anyone tried the new **TypeScript 5.4** features? The ` + "`NoInfer`" + ` utility type is a game changer:

` + "```typescript" + `
function createStore<T>(initial: NoInfer<T>) {
  let state = initial;

  return {
    get: () => state,
    set: (value: T) => { state = value; }
  };
}

// Now TypeScript won't infer from 'initial'
const store = createStore<number>(0);
` + "```" + `

This prevents accidental type widening. Really useful for generic functions!`,
				`## Quick SQL Tip

When dealing with large tables, always use ` + "`EXPLAIN ANALYZE`" + `:

` + "```sql" + `
EXPLAIN ANALYZE
SELECT u.name, COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.created_at > '2024-01-01'
GROUP BY u.id, u.name
HAVING COUNT(o.id) > 5
ORDER BY order_count DESC
LIMIT 100;
` + "```" + `

This shows you the actual execution plan and timing. *Much better than guessing!*`,
				`Here's my terminal setup for productivity:

` + "```bash" + `
# .zshrc aliases
alias gs="git status"
alias gc="git commit -m"
alias gp="git push"
alias gl="git log --oneline -20"

# Better ls
alias ll="ls -lah"
alias la="ls -A"

# Quick navigation
alias ..="cd .."
alias ...="cd ../.."
alias dev="cd ~/Developer"

# Docker shortcuts
alias dc="docker-compose"
alias dps="docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
` + "```" + `

Small things that save *hours* over time!`,
				"That's all great stuff! Thanks for sharing everyone. Looking forward to more discussions! 🚀",
			}

			for i, content := range sampleMessages {
				author := createdUsers[i%len(createdUsers)]
				messagesSvc.Create(ctx, author.ID, &messages.CreateIn{
					ChannelID: general.ID,
					Content:   content,
				})
			}
		}
	}

	// Summary
	ui.Summary([][2]string{
		{"Users", fmt.Sprintf("%d", len(createdUsers))},
		{"Servers", fmt.Sprintf("%d", len(sampleServers))},
	})

	ui.Success("Database seeded successfully")
	return nil
}

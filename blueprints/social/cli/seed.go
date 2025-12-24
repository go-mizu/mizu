package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/relationships"
	"github.com/go-mizu/blueprints/social/store/duckdb"
)

var (
	seedUsers int
	seedPosts int
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with sample data",
	Long:  `Seed the Social database with sample users and posts.`,
	RunE:  runSeed,
}

func init() {
	seedCmd.Flags().IntVar(&seedUsers, "users", 10, "Number of users to create")
	seedCmd.Flags().IntVar(&seedPosts, "posts", 50, "Number of posts to create")
}

func runSeed(cmd *cobra.Command, args []string) error {
	ui := NewUI()
	ui.Header("Social", Version)

	ui.Info("Seeding database...")
	ui.Item("Users", fmt.Sprintf("%d", seedUsers))
	ui.Item("Posts", fmt.Sprintf("%d", seedPosts))

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(dataDir, "social.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Create stores
	coreStore, err := duckdb.New(db)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	if err := coreStore.Ensure(context.Background()); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	accountsStore := duckdb.NewAccountsStore(db)
	postsStore := duckdb.NewPostsStore(db)
	relationshipsStore := duckdb.NewRelationshipsStore(db)
	interactionsStore := duckdb.NewInteractionsStore(db)

	// Create services
	accountsSvc := accounts.NewService(accountsStore, relationshipsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc, interactionsStore)
	relationshipsSvc := relationships.NewService(relationshipsStore, func(ctx context.Context, id string) (bool, error) {
		return false, nil
	})

	ctx := context.Background()

	// Create sample users
	usernames := []string{"alice", "bob", "charlie", "diana", "eve", "frank", "grace", "henry", "iris", "jack"}
	bios := []string{
		"Just a person who loves tech",
		"Coffee enthusiast",
		"Building the future",
		"Cat lover and coder",
		"Making the world better",
		"Photography and travel",
		"Open source contributor",
		"Startup founder",
		"Designer by day, gamer by night",
		"Music and movies",
	}

	var createdUsers []*accounts.Account
	for i := 0; i < seedUsers && i < len(usernames); i++ {
		username := usernames[i]
		bio := bios[i%len(bios)]

		account, err := accountsSvc.Create(ctx, &accounts.CreateIn{
			Username:    username,
			Email:       fmt.Sprintf("%s@example.com", username),
			Password:    "password123",
			DisplayName: fmt.Sprintf("%s User", username),
		})
		if err != nil {
			ui.Warning(fmt.Sprintf("Failed to create user %s: %v", username, err))
			continue
		}

		// Update bio
		bioStr := bio
		_, _ = accountsSvc.Update(ctx, account.ID, &accounts.UpdateIn{
			Bio: &bioStr,
		})

		createdUsers = append(createdUsers, account)
		ui.Item("Created", fmt.Sprintf("@%s", username))
	}

	// Create follow relationships
	for i, user := range createdUsers {
		for j, other := range createdUsers {
			if i != j && (i+j)%3 == 0 {
				_, _ = relationshipsSvc.Follow(ctx, user.ID, other.ID)
			}
		}
	}
	ui.Info("Created follow relationships")

	// Create sample posts
	postContents := []string{
		"Hello, world! This is my first post.",
		"Just discovered this amazing new platform!",
		"What a beautiful day to build something new.",
		"Coffee and code - the perfect combination.",
		"Thinking about the future of social media...",
		"Just shipped a new feature!",
		"Weekend vibes",
		"Learning something new every day.",
		"Open source is the way forward.",
		"Let's connect and share ideas!",
	}

	postsCreated := 0
	for postsCreated < seedPosts && len(createdUsers) > 0 {
		user := createdUsers[postsCreated%len(createdUsers)]
		content := postContents[postsCreated%len(postContents)]

		if postsCreated > 0 {
			content = fmt.Sprintf("%s #post%d", content, postsCreated)
		}

		_, err := postsSvc.Create(ctx, user.ID, &posts.CreateIn{
			Content: content,
		})
		if err != nil {
			ui.Warning(fmt.Sprintf("Failed to create post: %v", err))
		} else {
			postsCreated++
		}
	}
	ui.Item("Posts", fmt.Sprintf("%d created", postsCreated))

	ui.Success("Database seeded successfully")

	return nil
}

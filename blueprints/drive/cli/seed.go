package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

func newSeedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed demo data",
		RunE:  runSeed,
	}
}

func runSeed(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	store, err := duckdb.Open(dataDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	accountsSvc := accounts.NewService(store.Accounts())

	// Create demo user
	user, err := accountsSvc.Register(ctx, &accounts.RegisterIn{
		Username:    "demo",
		Email:       "demo@example.com",
		Password:    "demo1234",
		DisplayName: "Demo User",
	})
	if err != nil {
		if err == accounts.ErrUsernameTaken || err == accounts.ErrEmailTaken {
			fmt.Println("Demo user already exists")
		} else {
			return fmt.Errorf("create demo user: %w", err)
		}
	} else {
		fmt.Printf("Created demo user: %s (password: demo1234)\n", user.Username)
	}

	// Create admin user
	admin, err := accountsSvc.Register(ctx, &accounts.RegisterIn{
		Username:    "admin",
		Email:       "admin@example.com",
		Password:    "admin1234",
		DisplayName: "Administrator",
	})
	if err != nil {
		if err == accounts.ErrUsernameTaken || err == accounts.ErrEmailTaken {
			fmt.Println("Admin user already exists")
		} else {
			return fmt.Errorf("create admin user: %w", err)
		}
	} else {
		fmt.Printf("Created admin user: %s (password: admin1234)\n", admin.Username)
	}

	fmt.Println("Seed completed successfully")
	return nil
}

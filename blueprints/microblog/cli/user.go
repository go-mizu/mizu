package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// NewUser creates the user command group.
func NewUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage user accounts",
		Long:  `Create, list, and manage user accounts.`,
	}

	cmd.AddCommand(
		newUserCreate(),
		newUserList(),
		newUserVerify(),
		newUserSuspend(),
	)

	return cmd
}

func newUserCreate() *cobra.Command {
	var email string
	var password string
	var admin bool

	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a new user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			if password == "" {
				// Generate a random password if not provided
				password = fmt.Sprintf("temp%d", os.Getpid())
				fmt.Printf("Generated temporary password: %s\n", password)
			}

			store, cleanup, err := openAccountsStore()
			if err != nil {
				return err
			}
			defer cleanup()

			svc := accounts.NewService(store)
			account, err := svc.Create(context.Background(), &accounts.CreateIn{
				Username: username,
				Email:    email,
				Password: password,
			})
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}

			if admin {
				if err := svc.SetAdmin(context.Background(), account.ID, true); err != nil {
					return fmt.Errorf("set admin: %w", err)
				}
			}

			fmt.Printf("Created user: @%s (ID: %s)\n", account.Username, account.ID)
			if admin {
				fmt.Println("  Admin: yes")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&password, "password", "", "Password (generated if not provided)")
	cmd.Flags().BoolVar(&admin, "admin", false, "Create as admin")

	return cmd
}

func newUserList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, cleanup, err := openAccountsStore()
			if err != nil {
				return err
			}
			defer cleanup()

			svc := accounts.NewService(store)
			list, err := svc.List(context.Background(), 100, 0)
			if err != nil {
				return fmt.Errorf("list users: %w", err)
			}

			fmt.Printf("Users (%d total):\n\n", list.Total)
			for _, a := range list.Accounts {
				verified := ""
				if a.Verified {
					verified = " [verified]"
				}
				fmt.Printf("  @%-20s %s%s\n", a.Username, a.DisplayName, verified)
			}

			return nil
		},
	}

	return cmd
}

func newUserVerify() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <username>",
		Short: "Verify a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			store, cleanup, err := openAccountsStore()
			if err != nil {
				return err
			}
			defer cleanup()

			svc := accounts.NewService(store)
			account, err := svc.GetByUsername(context.Background(), username)
			if err != nil {
				return fmt.Errorf("find user: %w", err)
			}

			if err := svc.Verify(context.Background(), account.ID, true); err != nil {
				return fmt.Errorf("verify user: %w", err)
			}

			fmt.Printf("Verified user: @%s\n", account.Username)
			return nil
		},
	}

	return cmd
}

func newUserSuspend() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "suspend <username>",
		Short: "Suspend a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			store, cleanup, err := openAccountsStore()
			if err != nil {
				return err
			}
			defer cleanup()

			svc := accounts.NewService(store)
			account, err := svc.GetByUsername(context.Background(), username)
			if err != nil {
				return fmt.Errorf("find user: %w", err)
			}

			if err := svc.Suspend(context.Background(), account.ID, true); err != nil {
				return fmt.Errorf("suspend user: %w", err)
			}

			fmt.Printf("Suspended user: @%s\n", account.Username)
			if reason != "" {
				fmt.Printf("  Reason: %s\n", reason)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Suspension reason")

	return cmd
}

func openAccountsStore() (accounts.Store, func(), error) {
	dbPath := filepath.Join(dataDir, "microblog.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema using core store
	coreStore, err := duckdb.New(db)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("create store: %w", err)
	}

	if err := coreStore.Ensure(context.Background()); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Return accounts-specific store
	accountsStore := duckdb.NewAccountsStore(db)

	cleanup := func() {
		db.Close()
	}

	return accountsStore, cleanup, nil
}

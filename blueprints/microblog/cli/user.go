package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
			ui := NewUI()
			start := time.Now()

			ui.Header(iconUser, "Create User")
			ui.Blank()

			generatedPassword := false
			if password == "" {
				password = fmt.Sprintf("temp%d", os.Getpid())
				generatedPassword = true
			}

			store, cleanup, err := openAccountsStore()
			if err != nil {
				ui.Error("Failed to open database")
				return err
			}
			defer cleanup()

			ui.StartSpinner("Creating user account...")
			svc := accounts.NewService(store)
			account, err := svc.Create(context.Background(), &accounts.CreateIn{
				Username: username,
				Email:    email,
				Password: password,
			})
			if err != nil {
				ui.StopSpinnerError("Failed to create user")
				return fmt.Errorf("create user: %w", err)
			}
			ui.StopSpinner("Account created", time.Since(start))

			if admin {
				if err := svc.SetAdmin(context.Background(), account.ID, true); err != nil {
					ui.Warn("Failed to set admin flag")
				}
			}

			ui.Success("User created successfully")
			ui.Blank()
			ui.Info("Username", usernameStyle.Render("@"+account.Username))
			ui.Info("ID", account.ID)
			if email != "" {
				ui.Info("Email", email)
			}
			if admin {
				ui.Info("Admin", successStyle.Render("yes"))
			}
			if generatedPassword {
				ui.Blank()
				ui.Warn("Generated temporary password:")
				ui.Hint(password)
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
			ui := NewUI()

			store, cleanup, err := openAccountsStore()
			if err != nil {
				ui.Error("Failed to open database")
				return err
			}
			defer cleanup()

			ui.Header(iconUser, "User Accounts")
			ui.Blank()

			svc := accounts.NewService(store)
			list, err := svc.List(context.Background(), 100, 0)
			if err != nil {
				ui.Error("Failed to list users")
				return fmt.Errorf("list users: %w", err)
			}

			if list.Total == 0 {
				ui.Hint("No users found. Use 'microblog user create' to add users.")
				return nil
			}

			ui.Info("Total users", fmt.Sprintf("%d", list.Total))
			ui.Blank()

			for _, a := range list.Accounts {
				ui.UserRow(a.Username, a.DisplayName, a.Verified, a.Admin, a.Suspended)
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
			ui := NewUI()

			store, cleanup, err := openAccountsStore()
			if err != nil {
				ui.Error("Failed to open database")
				return err
			}
			defer cleanup()

			ui.StartSpinner(fmt.Sprintf("Verifying @%s...", username))

			svc := accounts.NewService(store)
			account, err := svc.GetByUsername(context.Background(), username)
			if err != nil {
				ui.StopSpinnerError("User not found")
				return fmt.Errorf("find user: %w", err)
			}

			if err := svc.Verify(context.Background(), account.ID, true); err != nil {
				ui.StopSpinnerError("Failed to verify user")
				return fmt.Errorf("verify user: %w", err)
			}

			ui.StopSpinnerError("") // Clear line
			ui.Success(fmt.Sprintf("Verified user %s", usernameStyle.Render("@"+account.Username)))

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
			ui := NewUI()

			store, cleanup, err := openAccountsStore()
			if err != nil {
				ui.Error("Failed to open database")
				return err
			}
			defer cleanup()

			ui.StartSpinner(fmt.Sprintf("Suspending @%s...", username))

			svc := accounts.NewService(store)
			account, err := svc.GetByUsername(context.Background(), username)
			if err != nil {
				ui.StopSpinnerError("User not found")
				return fmt.Errorf("find user: %w", err)
			}

			if err := svc.Suspend(context.Background(), account.ID, true); err != nil {
				ui.StopSpinnerError("Failed to suspend user")
				return fmt.Errorf("suspend user: %w", err)
			}

			ui.StopSpinnerError("") // Clear line
			ui.Success(fmt.Sprintf("Suspended user %s", usernameStyle.Render("@"+account.Username)))
			if reason != "" {
				ui.Info("Reason", reason)
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

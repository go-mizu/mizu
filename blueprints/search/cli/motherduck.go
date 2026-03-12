package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/motherduck"
	"github.com/spf13/cobra"
)

// NewMotherDuck returns the `search motherduck` command tree.
func NewMotherDuck() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "motherduck",
		Short: "Manage MotherDuck accounts and cloud DuckDB databases",
		Long: `Manage MotherDuck (cloud DuckDB) accounts, databases, and run SQL queries.

Registration uses the motherduck-tool binary (browser automation).
Account management and queries are pure Go via the DuckDB md: protocol.

Build the binary first:
  cd blueprints/search/tools/motherduck && make install`,
	}
	cmd.AddCommand(newMDRegister())
	cmd.AddCommand(newMDAccount())
	cmd.AddCommand(newMDDB())
	cmd.AddCommand(newMDQuery())
	return cmd
}

// ---- register ---------------------------------------------------------------

func newMDRegister() *cobra.Command {
	var noHeadless bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Auto-register a MotherDuck account via browser automation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDRegister(noHeadless, verbose)
		},
	}
	cmd.Flags().BoolVar(&noHeadless, "no-headless", false, "Show browser window")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose browser output")
	return cmd
}

func runMDRegister(noHeadless, verbose bool) error {
	bin := findMDToolBinary()
	if bin == "" {
		return fmt.Errorf("motherduck-tool binary not found\nBuild it: cd blueprints/search/tools/motherduck && make install")
	}

	args := []string{"register", "--json"}
	if noHeadless {
		args = append(args, "--no-headless")
	}
	if verbose {
		args = append(args, "--verbose")
	}

	var stdout bytes.Buffer
	proc := exec.Command(bin, args...)
	proc.Stderr = os.Stderr
	proc.Stdout = &stdout

	fmt.Fprintln(os.Stderr, subtitleStyle.Render("Running: "+bin+" "+strings.Join(args, " ")))
	if err := proc.Run(); err != nil {
		return fmt.Errorf("motherduck-tool failed: %w", err)
	}

	var result motherduck.RegisterResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return fmt.Errorf("invalid JSON from motherduck-tool: %w\nOutput: %s", err, stdout.String())
	}

	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if err := store.AddAccount(result); err != nil {
		return fmt.Errorf("store account: %w", err)
	}

	fmt.Println(successStyle.Render("Registered: " + result.Email))
	fmt.Println(subtitleStyle.Render("Token:      " + result.Token[:20] + "..."))
	fmt.Println(subtitleStyle.Render("Stored in:  " + motherduck.DefaultDBPath()))
	return nil
}

// ---- account ----------------------------------------------------------------

func newMDAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage MotherDuck accounts",
	}
	cmd.AddCommand(newMDAccountLS())
	cmd.AddCommand(newMDAccountRM())
	return cmd
}

func newMDAccountLS() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDAccountLS()
		},
	}
}

func runMDAccountLS() error {
	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	accounts, err := store.ListAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		fmt.Println(warningStyle.Render("No accounts registered. Run: search motherduck register"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, titleStyle.Render("Email\tDatabases\tActive\tCreated"))
	fmt.Fprintln(w, strings.Repeat("─", 70))
	for _, a := range accounts {
		active := successStyle.Render("✓")
		if !a.IsActive {
			active = errorStyle.Render("✗")
		}
		created := a.CreatedAt
		if len(created) > 16 {
			created = created[:16]
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", a.Email, a.DBCount, active, created)
	}
	return w.Flush()
}

func newMDAccountRM() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <email>",
		Short: "Deactivate an account (local only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDAccountRM(args[0])
		},
	}
}

func runMDAccountRM(email string) error {
	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.DeactivateAccount(email); err != nil {
		return err
	}
	fmt.Println(warningStyle.Render("Deactivated: " + email))
	return nil
}

// ---- db ---------------------------------------------------------------------

func newMDDB() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage MotherDuck databases",
	}
	cmd.AddCommand(newMDDBCreate())
	cmd.AddCommand(newMDDBLS())
	cmd.AddCommand(newMDDBUse())
	cmd.AddCommand(newMDDBRM())
	return cmd
}

func newMDDBCreate() *cobra.Command {
	var alias string
	var account string
	var setDefault bool

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new database on MotherDuck",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDDBCreate(args[0], alias, account, setDefault)
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "Local alias (default: same as name)")
	cmd.Flags().StringVar(&account, "account", "", "Account email to use")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set as default database")
	return cmd
}

func runMDDBCreate(name, alias, accountEmail string, setDefault bool) error {
	if alias == "" {
		alias = name
	}

	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	var acc *motherduck.Account
	if accountEmail != "" {
		a, err := store.GetDatabaseByAlias(accountEmail)
		if err != nil {
			return err
		}
		if a == nil {
			return fmt.Errorf("account not found: %s", accountEmail)
		}
		acc = &motherduck.Account{ID: a.AccountID, Token: a.Token}
	} else {
		acc, err = store.GetFirstActiveAccount()
		if err != nil {
			return err
		}
		if acc == nil {
			return fmt.Errorf("no active accounts — run: search motherduck register")
		}
	}

	fmt.Printf("Creating database '%s' on MotherDuck...\n", name)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := motherduck.CreateDB(ctx, acc.Token, name); err != nil {
		return fmt.Errorf("create db: %w", err)
	}

	if err := store.AddDatabase(acc.ID, name, alias); err != nil {
		return fmt.Errorf("store db: %w", err)
	}
	if setDefault {
		if err := store.SetDefault(alias); err != nil {
			return err
		}
	}

	fmt.Println(successStyle.Render("Created: " + name + " (alias: " + alias + ")"))
	if setDefault {
		fmt.Println(successStyle.Render("Set as default."))
	}
	return nil
}

func newMDDBLS() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDDBLS()
		},
	}
}

func runMDDBLS() error {
	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	dbs, err := store.ListDatabases()
	if err != nil {
		return err
	}
	if len(dbs) == 0 {
		fmt.Println(warningStyle.Render("No databases. Run: search motherduck db create <name>"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, titleStyle.Render("Alias\tName\tAccount\tDef\tQueries\tLast Used\tCreated"))
	fmt.Fprintln(w, strings.Repeat("─", 90))
	for _, d := range dbs {
		def := ""
		if d.IsDefault {
			def = successStyle.Render("●")
		}
		last := d.LastUsedAt
		if len(last) > 16 {
			last = last[:16]
		}
		created := d.CreatedAt
		if len(created) > 16 {
			created = created[:16]
		}
		alias := d.Alias
		if d.IsDefault {
			alias = titleStyle.Render(alias)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			alias, d.Name, d.Email, def, d.QueryCount, last, created)
	}
	return w.Flush()
}

func newMDDBUse() *cobra.Command {
	return &cobra.Command{
		Use:   "use <alias>",
		Short: "Set the default database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDDBUse(args[0])
		},
	}
}

func runMDDBUse(alias string) error {
	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.SetDefault(alias); err != nil {
		return err
	}
	fmt.Println(successStyle.Render("Default set to: " + alias))
	return nil
}

func newMDDBRM() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <alias>",
		Short: "Remove a database from local state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDDBRM(args[0])
		},
	}
}

func runMDDBRM(alias string) error {
	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.RemoveDatabase(alias); err != nil {
		return err
	}
	fmt.Println(warningStyle.Render("Removed: " + alias + " (local state only)"))
	return nil
}

// ---- query ------------------------------------------------------------------

func newMDQuery() *cobra.Command {
	var dbAlias string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "query <sql>",
		Short: "Run SQL against a MotherDuck database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMDQuery(args[0], dbAlias, jsonOut)
		},
	}
	cmd.Flags().StringVar(&dbAlias, "db", "", "Database alias (default: current default)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")
	return cmd
}

func runMDQuery(sqlStr, dbAlias string, jsonOut bool) error {
	store, err := motherduck.NewStore(motherduck.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	var db *motherduck.Database
	if dbAlias != "" {
		db, err = store.GetDatabaseByAlias(dbAlias)
		if err != nil {
			return err
		}
		if db == nil {
			return fmt.Errorf("no database with alias: %s", dbAlias)
		}
	} else {
		db, err = store.GetDefaultDatabase()
		if err != nil {
			return err
		}
		if db == nil {
			return fmt.Errorf("no default database — run: search motherduck db use <alias>")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t0 := time.Now()
	rows, cols, err := motherduck.Query(ctx, db.Token, db.Name, sqlStr)
	elapsed := time.Since(t0)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	_ = store.LogQuery(db.ID, sqlStr, len(rows), int(elapsed.Milliseconds()))
	_ = store.TouchLastUsed(db.Alias)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	if len(rows) == 0 {
		fmt.Println(subtitleStyle.Render("No rows returned."))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(cols, "\t"))
	fmt.Fprintln(w, strings.Repeat("─", 60))
	for _, row := range rows {
		vals := make([]string, len(cols))
		for i, col := range cols {
			v := row[col]
			if v == nil {
				vals[i] = "NULL"
			} else {
				vals[i] = fmt.Sprintf("%v", v)
			}
		}
		fmt.Fprintln(w, strings.Join(vals, "\t"))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Printf("\n%s\n", subtitleStyle.Render(
		fmt.Sprintf("%d row(s) · %dms · db: %s", len(rows), elapsed.Milliseconds(), db.Name),
	))
	return nil
}

// ---- helpers ----------------------------------------------------------------

func findMDToolBinary() string {
	if p := os.Getenv("MOTHERDUCK_TOOL"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "bin", "motherduck-tool")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if p, err := exec.LookPath("motherduck-tool"); err == nil {
		return p
	}
	return ""
}

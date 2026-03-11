package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/clickhouse"
	"github.com/spf13/cobra"
)

// NewClickHouse returns the `search clickhouse` command tree.
func NewClickHouse() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clickhouse",
		Short: "Manage ClickHouse Cloud accounts and run queries",
		Long: `Manage ClickHouse Cloud accounts, services, and run SQL queries.

Registration uses the clickhouse-tool binary (browser automation).
Account management and queries are pure Go.

Build the binary first:
  cd blueprints/search/tools/clickhouse && make install`,
	}
	cmd.AddCommand(newCHRegister())
	cmd.AddCommand(newCHAccount())
	cmd.AddCommand(newCHService())
	cmd.AddCommand(newCHQuery())
	return cmd
}

// ---- register ---------------------------------------------------------------

func newCHRegister() *cobra.Command {
	var noHeadless bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Auto-register a ClickHouse Cloud account via browser automation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHRegister(noHeadless, verbose)
		},
	}
	cmd.Flags().BoolVar(&noHeadless, "no-headless", false, "Show browser window")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose browser output")
	return cmd
}

func runCHRegister(noHeadless, verbose bool) error {
	bin := findCHToolBinary()
	if bin == "" {
		return fmt.Errorf("clickhouse-tool binary not found\nBuild it: cd blueprints/search/tools/clickhouse && make install")
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
		return fmt.Errorf("clickhouse-tool failed: %w", err)
	}

	var result clickhouse.RegisterResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return fmt.Errorf("invalid JSON from clickhouse-tool: %w\nOutput: %s", err, stdout.String())
	}

	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if err := store.AddAccount(result); err != nil {
		return fmt.Errorf("store account: %w", err)
	}

	fmt.Println(successStyle.Render("Registered: " + result.Email))
	if result.Host != "" {
		fmt.Println(subtitleStyle.Render("Service:    " + result.Host))
	}
	fmt.Println(subtitleStyle.Render("Stored in:  " + clickhouse.DefaultDBPath()))
	return nil
}

// ---- account ----------------------------------------------------------------

func newCHAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage ClickHouse accounts",
	}
	cmd.AddCommand(newCHAccountLS())
	cmd.AddCommand(newCHAccountRM())
	return cmd
}

func newCHAccountLS() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHAccountLS()
		},
	}
}

func runCHAccountLS() error {
	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	accounts, err := store.ListAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		fmt.Println(warningStyle.Render("No accounts registered. Run: search clickhouse register"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, titleStyle.Render("Email\tOrg ID\tServices\tActive\tCreated"))
	fmt.Fprintln(w, strings.Repeat("─", 70))
	for _, a := range accounts {
		active := successStyle.Render("✓")
		if !a.IsActive {
			active = errorStyle.Render("✗")
		}
		orgShort := a.OrgID
		if len(orgShort) > 16 {
			orgShort = orgShort[:16] + "…"
		}
		created := a.CreatedAt
		if len(created) > 16 {
			created = created[:16]
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", a.Email, orgShort, a.SvcCount, active, created)
	}
	return w.Flush()
}

func newCHAccountRM() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <email>",
		Short: "Deactivate an account (local only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHAccountRM(args[0])
		},
	}
}

func runCHAccountRM(email string) error {
	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
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

// ---- service ----------------------------------------------------------------

func newCHService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage ClickHouse services",
	}
	cmd.AddCommand(newCHServiceLS())
	cmd.AddCommand(newCHServiceUse())
	cmd.AddCommand(newCHServiceRM())
	return cmd
}

func newCHServiceLS() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHServiceLS()
		},
	}
}

func runCHServiceLS() error {
	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	services, err := store.ListServices()
	if err != nil {
		return err
	}
	if len(services) == 0 {
		fmt.Println(warningStyle.Render("No services. Run: search clickhouse register"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, titleStyle.Render("Alias\tName\tHost\tAccount\tDef\tQueries\tLast Used"))
	fmt.Fprintln(w, strings.Repeat("─", 90))
	for _, sv := range services {
		def := ""
		if sv.IsDefault {
			def = successStyle.Render("●")
		}
		host := sv.Host
		if len(host) > 35 {
			host = host[:35] + "…"
		}
		last := sv.LastUsedAt
		if len(last) > 16 {
			last = last[:16]
		}
		alias := sv.Alias
		if sv.IsDefault {
			alias = titleStyle.Render(alias)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			alias, sv.Name, host, sv.Email, def, sv.QueryCount, last)
	}
	return w.Flush()
}

func newCHServiceUse() *cobra.Command {
	return &cobra.Command{
		Use:   "use <alias>",
		Short: "Set the default service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHServiceUse(args[0])
		},
	}
}

func runCHServiceUse(alias string) error {
	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
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

func newCHServiceRM() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <alias>",
		Short: "Remove a service from local state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHServiceRM(args[0])
		},
	}
}

func runCHServiceRM(alias string) error {
	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.RemoveService(alias); err != nil {
		return err
	}
	fmt.Println(warningStyle.Render("Removed: " + alias + " (local state only)"))
	return nil
}

// ---- query ------------------------------------------------------------------

func newCHQuery() *cobra.Command {
	var serviceAlias string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "query <sql>",
		Short: "Run SQL against a ClickHouse Cloud service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCHQuery(args[0], serviceAlias, jsonOut)
		},
	}
	cmd.Flags().StringVar(&serviceAlias, "service", "", "Service alias (default: current default)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")
	return cmd
}

func runCHQuery(sqlStr, serviceAlias string, jsonOut bool) error {
	store, err := clickhouse.NewStore(clickhouse.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	var svc *clickhouse.Service
	if serviceAlias != "" {
		svc, err = store.GetServiceByAlias(serviceAlias)
		if err != nil {
			return err
		}
		if svc == nil {
			return fmt.Errorf("no service with alias: %s", serviceAlias)
		}
	} else {
		svc, err = store.GetDefaultService()
		if err != nil {
			return err
		}
		if svc == nil {
			return fmt.Errorf("no default service — run: search clickhouse service use <alias>")
		}
	}

	client := clickhouse.NewClient(svc.Host, svc.Port, svc.DBUser, svc.DBPassword)

	t0 := time.Now()
	rows, cols, err := client.Query(sqlStr)
	elapsed := time.Since(t0)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	_ = store.LogQuery(svc.ID, sqlStr, len(rows), int(elapsed.Milliseconds()))
	_ = store.TouchLastUsed(svc.Alias)

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
		fmt.Sprintf("%d row(s) · %dms · service: %s", len(rows), elapsed.Milliseconds(), svc.Name),
	))
	return nil
}

// ---- helpers ----------------------------------------------------------------

func findCHToolBinary() string {
	if p := os.Getenv("CLICKHOUSE_TOOL"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "bin", "clickhouse-tool")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if p, err := exec.LookPath("clickhouse-tool"); err == nil {
		return p
	}
	return ""
}

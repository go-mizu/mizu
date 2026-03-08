package cli

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
	"github.com/spf13/cobra"
)

func NewSerp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serp",
		Short: "SerpAPI account management and search",
	}
	cmd.AddCommand(newSerpRegister(), newSerpSearch(), newSerpList(), newSerpStatus())
	return cmd
}

func newSerpRegister() *cobra.Command {
	var count int
	var verbose bool
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register SerpAPI accounts via mail.tm",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			for i := 0; i < count; i++ {
				fmt.Printf("[%d/%d] registering...\n", i+1, count)
				acc, err := serp.RegisterAccount(serp.RegisterOptions{Verbose: verbose})
				if err != nil {
					fmt.Fprintf(os.Stderr, "  error: %v\n", err)
					continue
				}
				store.Add(*acc)
				if err := store.Save(); err != nil {
					return err
				}
				fmt.Printf("  OK: %s (searches_left=%d)\n", acc.Email, acc.SearchesLeft)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&count, "count", "n", 1, "number of accounts to register")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	return cmd
}

func newSerpSearch() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Google via SerpAPI using a random available key",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			// Prune exhausted (searches_left == 0 only)
			removed := store.PruneExhausted()
			if removed > 0 {
				fmt.Fprintf(os.Stderr, "removed %d exhausted account(s)\n", removed)
				store.Save()
			}
			available := store.Available()
			if len(available) == 0 {
				return fmt.Errorf("no available keys (searches_left >= 10); run `serp register`")
			}
			acc := available[rand.Intn(len(available))]
			fmt.Fprintf(os.Stderr, "using: %s (searches_left=%d)\n", acc.Email, acc.SearchesLeft)

			client := serp.NewSerpAPIClient()
			result, err := client.Search(acc.APIKey, query)
			if err != nil {
				return err
			}

			// Update searches_left from API (fallback: decrement locally)
			if info, err := client.GetAccount(acc.APIKey); err == nil {
				store.UpdateSearchesLeft(acc.APIKey, info.TotalSearchesLeft)
			} else {
				store.UpdateSearchesLeft(acc.APIKey, acc.SearchesLeft-1)
			}
			store.Save()

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			// Pretty print organic results
			for i, r := range result.OrganicResults {
				fmt.Printf("%d. %v\n   %v\n   %v\n\n", i+1, r["title"], r["link"], r["snippet"])
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output raw JSON")
	return cmd
}

func newSerpList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored SerpAPI accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			if len(store.Accounts) == 0 {
				fmt.Println("no accounts stored")
				return nil
			}
			fmt.Printf("%-30s %-14s %-20s\n", "EMAIL", "SEARCHES_LEFT", "LAST_CHECKED")
			for _, a := range store.Accounts {
				status := fmt.Sprintf("%d", a.SearchesLeft)
				if a.SearchesLeft < 10 {
					status += " (low)"
				}
				checked := "never"
				if !a.LastChecked.IsZero() {
					checked = a.LastChecked.Format("2006-01-02 15:04")
				}
				fmt.Printf("%-30s %-14s %-20s\n", a.Email, status, checked)
			}
			return nil
		},
	}
}

func newSerpStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Refresh searches_left for all stored accounts from SerpAPI",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			client := serp.NewSerpAPIClient()
			for _, a := range store.Accounts {
				info, err := client.GetAccount(a.APIKey)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  %s: error: %v\n", a.Email, err)
					continue
				}
				store.UpdateSearchesLeft(a.APIKey, info.TotalSearchesLeft)
				fmt.Printf("  %s: searches_left=%d\n", a.Email, info.TotalSearchesLeft)
			}
			return store.Save()
		},
	}
}

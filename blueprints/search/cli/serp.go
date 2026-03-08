package cli

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
	"github.com/spf13/cobra"
)

func NewSerp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serp",
		Short: "SERP API management and search",
	}
	cmd.AddCommand(
		newSerpRegister(),
		newSerpSignup(),
		newSerpSearch(),
		newSerpList(),
		newSerpStatus(),
		newSerpAddKey(),
		newSerpTest(),
	)
	return cmd
}

func newSerpAddKey() *cobra.Command {
	return &cobra.Command{
		Use:   "add-key <provider> <api-key>",
		Short: "Add an API key for a SERP provider",
		Long: `Add an API key for a SERP provider. Supported providers:
  serper    - serper.dev (2,500 free queries)
  zenserp   - zenserp.com (50 free/month)
  searchapi - searchapi.io (100 free queries)
  serpstack - serpstack.com (100 free/month)
  serply    - serply.io (free monthly credits)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, apiKey := args[0], args[1]
			providers := serp.AllProviders()
			if _, ok := providers[provider]; !ok {
				names := make([]string, 0, len(providers))
				for k := range providers {
					names = append(names, k)
				}
				sort.Strings(names)
				return fmt.Errorf("unknown provider %q; supported: %s", provider, strings.Join(names, ", "))
			}

			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			store.Add(serp.Account{
				APIKey:       apiKey,
				Provider:     provider,
				RegisteredAt: time.Now(),
				SearchesLeft: 9999, // unknown, set high
			})
			if err := store.Save(); err != nil {
				return err
			}
			fmt.Printf("added %s key: %s...%s\n", provider, apiKey[:4], apiKey[len(apiKey)-4:])
			return nil
		},
	}
}

func newSerpTest() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "test <provider> <api-key>",
		Short: "Test a SERP provider API key with a search query",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName, apiKey := args[0], args[1]
			providers := serp.AllProviders()
			p, ok := providers[providerName]
			if !ok {
				names := make([]string, 0, len(providers))
				for k := range providers {
					names = append(names, k)
				}
				sort.Strings(names)
				return fmt.Errorf("unknown provider %q; supported: %s", providerName, strings.Join(names, ", "))
			}

			fmt.Printf("testing %s with query %q...\n", providerName, query)
			result, err := p.Search(apiKey, query)
			if err != nil {
				return fmt.Errorf("%s FAILED: %w", providerName, err)
			}
			fmt.Printf("%s OK — %d results\n", providerName, len(result.OrganicResults))
			for i, r := range result.OrganicResults {
				if i >= 5 {
					fmt.Printf("  ... and %d more\n", len(result.OrganicResults)-5)
					break
				}
				fmt.Printf("  %d. %v\n     %v\n", i+1, r["title"], r["link"])
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "golang programming", "search query for testing")
	return cmd
}

func newSerpSignup() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "signup <provider>",
		Short: "Auto-signup for a SERP provider (serper, zenserp, searchapi, serpstack, serply)",
		Long: `Auto-signup for a SERP API provider using a temporary mail.tm email.
Opens a browser to fill the signup form. After registration, extracts the
API key and stores it locally.

Examples:
  search serp signup serper
  search serp signup searchapi --verbose`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if !serp.HasRegistrar(provider) {
				return fmt.Errorf("no auto-signup for provider %q; supported: jina, parallel, serper, zenserp, searchapi, serpstack, serply, firecrawl", provider)
			}

			fmt.Printf("signing up for %s...\n", provider)
			acc, err := serp.RegisterProvider(serp.ProviderRegisterOptions{
				Provider: provider,
				Verbose:  verbose,
			})
			if err != nil {
				return fmt.Errorf("signup failed: %w", err)
			}

			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			store.Add(*acc)
			if err := store.Save(); err != nil {
				return err
			}

			if acc.APIKey == "" {
				fmt.Printf("account created for %s (email: %s)\n", provider, acc.Email)
				fmt.Println("API key not yet available — complete onboarding manually, then:")
				fmt.Printf("  search serp add-key %s <your-api-key>\n", provider)
				return nil
			}

			fmt.Printf("OK! %s key: %s...%s\n", provider, acc.APIKey[:4], acc.APIKey[len(acc.APIKey)-4:])

			// Quick test
			fmt.Printf("testing with query \"test\"...\n")
			providers := serp.AllProviders()
			p := providers[provider]
			result, err := p.Search(acc.APIKey, "test")
			if err != nil {
				fmt.Printf("WARNING: search test failed: %v\n", err)
			} else {
				fmt.Printf("search OK — %d results\n", len(result.OrganicResults))
			}

			return nil
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	return cmd
}

func newSerpRegister() *cobra.Command {
	var count int
	var verbose bool
	var proxy string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register SerpAPI accounts via mail.tm",
		Long: `Register SerpAPI accounts using temporary mail.tm emails.

Opens a browser to fill the SerpAPI signup form. You must solve the
reCAPTCHA manually (or set TWOCAPTCHA_KEY for auto-solving).

If SerpAPI blocks your IP, use --proxy to route through a different IP:
  search serp register --proxy socks5://host:port --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if proxy != "" {
				os.Setenv("SERP_PROXY", proxy)
			}
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
	cmd.Flags().StringVar(&proxy, "proxy", "", "proxy URL for browser (e.g. socks5://host:port)")
	return cmd
}

func newSerpSearch() *cobra.Command {
	var jsonOutput bool
	var providerName string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Google via SERP provider",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}

			providers := serp.AllProviders()

			// Find an account to use
			var acc serp.Account
			var provider serp.Provider
			if providerName != "" {
				p, ok := providers[providerName]
				if !ok {
					return fmt.Errorf("unknown provider %q", providerName)
				}
				accs := store.ByProvider(providerName)
				if len(accs) == 0 {
					return fmt.Errorf("no keys for provider %q; run `serp add-key %s <key>`", providerName, providerName)
				}
				acc = accs[rand.Intn(len(accs))]
				provider = p
			} else {
				// Try providers in priority order
				order := []string{"jina", "parallel", "serper", "searchapi", "zenserp", "serpstack", "serply", "firecrawl"}
				for _, name := range order {
					accs := store.ByProvider(name)
					if len(accs) > 0 {
						acc = accs[rand.Intn(len(accs))]
						provider = providers[name]
						break
					}
				}
				// Fallback to legacy serpapi accounts (no provider field)
				if provider == nil {
					available := store.Available()
					if len(available) > 0 {
						acc = available[rand.Intn(len(available))]
						client := serp.NewSerpAPIClient()
						result, err := client.Search(acc.APIKey, query)
						if err != nil {
							return err
						}
						if jsonOutput {
							enc := json.NewEncoder(os.Stdout)
							enc.SetIndent("", "  ")
							return enc.Encode(result)
						}
						for i, r := range result.OrganicResults {
							fmt.Printf("%d. %v\n   %v\n   %v\n\n", i+1, r["title"], r["link"], r["snippet"])
						}
						return nil
					}
					return fmt.Errorf("no API keys stored; run `serp add-key <provider> <key>`")
				}
			}

			fmt.Fprintf(os.Stderr, "using: %s (%s...)\n", acc.Provider, acc.APIKey[:8])
			result, err := provider.Search(acc.APIKey, query)
			if err != nil {
				return err
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			for i, r := range result.OrganicResults {
				fmt.Printf("%d. %v\n   %v\n   %v\n\n", i+1, r["title"], r["link"], r["snippet"])
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output raw JSON")
	cmd.Flags().StringVarP(&providerName, "provider", "p", "", "provider to use (serper, zenserp, searchapi, serpstack, serply)")
	return cmd
}

func newSerpList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored SERP API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			if len(store.Accounts) == 0 {
				fmt.Println("no accounts stored")
				return nil
			}
			fmt.Printf("%-12s %-30s %-14s %-20s\n", "PROVIDER", "KEY/EMAIL", "SEARCHES_LEFT", "LAST_CHECKED")
			for _, a := range store.Accounts {
				provider := a.Provider
				if provider == "" {
					provider = "serpapi"
				}
				label := a.Email
				if label == "" && len(a.APIKey) > 12 {
					label = a.APIKey[:8] + "..." + a.APIKey[len(a.APIKey)-4:]
				} else if label == "" {
					label = a.APIKey
				}
				status := fmt.Sprintf("%d", a.SearchesLeft)
				checked := "never"
				if !a.LastChecked.IsZero() {
					checked = a.LastChecked.Format("2006-01-02 15:04")
				}
				fmt.Printf("%-12s %-30s %-14s %-20s\n", provider, label, status, checked)
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
				if a.Provider != "" && a.Provider != "serpapi" {
					fmt.Printf("  %s (%s): skip (no status API)\n", a.Provider, a.APIKey[:8])
					continue
				}
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

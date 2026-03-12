package cli

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
	"github.com/go-mizu/mizu/blueprints/search/pkg/serp/jina"
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
		newSerpRotate(),
		newSerpInstall(),
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

			// Jina keys go to dedicated store
			if provider == "jina" {
				jinaStore, err := jina.LoadKeyStore(jina.DefaultKeyStorePath())
				if err != nil {
					return err
				}
				jinaStore.Add(apiKey)
				if err := jinaStore.Save(); err != nil {
					return err
				}
			} else {
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

			if acc.APIKey == "" {
				fmt.Printf("account created for %s (email: %s)\n", provider, acc.Email)
				fmt.Println("API key not yet available — complete onboarding manually, then:")
				fmt.Printf("  search serp add-key %s <your-api-key>\n", provider)
				return nil
			}

			// Save to dedicated Jina store if Jina provider
			if provider == "jina" {
				jinaStore, err := jina.LoadKeyStore(jina.DefaultKeyStorePath())
				if err != nil {
					return err
				}
				jinaStore.Add(acc.APIKey)
				if err := jinaStore.Save(); err != nil {
					return err
				}
			} else {
				// Save to legacy serp store for other providers
				store, err := serp.LoadStore(serp.DefaultStorePath())
				if err != nil {
					return err
				}
				store.Add(*acc)
				if err := store.Save(); err != nil {
					return err
				}
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
	var autoProvision bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Google via SERP provider",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			providers := serp.AllProviders()

			// Try Jina key store first ($HOME/data/jina/keys.json)
			jinaStore, err := jina.LoadKeyStore(jina.DefaultKeyStorePath())
			if err != nil {
				return err
			}

			var apiKey string
			var providerUsed string
			var provider serp.Provider

			if providerName == "" || providerName == "jina" {
				keys := jinaStore.Available()
				if len(keys) > 0 {
					k := keys[rand.Intn(len(keys))]
					apiKey = k.APIKey
					providerUsed = "jina"
					provider = providers["jina"]
				}
			}

			// If not Jina or no Jina keys, try other providers via legacy store
			if provider == nil && providerName != "jina" {
				store, err := serp.LoadStore(serp.DefaultStorePath())
				if err != nil {
					return err
				}
				if providerName != "" {
					p, ok := providers[providerName]
					if !ok {
						return fmt.Errorf("unknown provider %q", providerName)
					}
					accs := store.ByProvider(providerName)
					if len(accs) > 0 {
						acc := accs[rand.Intn(len(accs))]
						apiKey = acc.APIKey
						providerUsed = providerName
						provider = p
					}
				} else {
					order := []string{"parallel", "serper", "searchapi", "zenserp", "serpstack", "serply", "firecrawl"}
					for _, name := range order {
						accs := store.ByProvider(name)
						if len(accs) > 0 {
							acc := accs[rand.Intn(len(accs))]
							apiKey = acc.APIKey
							providerUsed = name
							provider = providers[name]
							break
						}
					}
				}
			}

			// Auto-provision Jina key if needed
			if provider == nil {
				if autoProvision {
					key, err := serpAutoProvisionJina(jinaStore)
					if err != nil {
						return err
					}
					apiKey = key
					providerUsed = "jina"
					provider = providers["jina"]
				} else {
					return fmt.Errorf("no API keys stored; run `serp add-key <provider> <key>` or use --auto to get a Jina key")
				}
			}

			fmt.Fprintf(os.Stderr, "using: %s (%s...)\n", providerUsed, apiKey[:8])
			result, err := provider.Search(apiKey, query)
			if err != nil {
				// If balance error, try to rotate
				if strings.Contains(err.Error(), "402") || strings.Contains(err.Error(), "InsufficientBalance") {
					fmt.Fprintf(os.Stderr, "key depleted, removing...\n")
					jinaStore.Remove(apiKey)
					_ = jinaStore.Save()
					if autoProvision {
						newKey, err := serpAutoProvisionJina(jinaStore)
						if err != nil {
							return err
						}
						result, err = provider.Search(newKey, query)
						if err != nil {
							return err
						}
					} else {
						return fmt.Errorf("key depleted; run with --auto to auto-provision a new one")
					}
				} else {
					return err
				}
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			renderSearchResults(query, providerUsed, result)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output raw JSON")
	cmd.Flags().StringVarP(&providerName, "provider", "p", "", "provider to use (jina, serper, zenserp, searchapi, etc.)")
	cmd.Flags().BoolVar(&autoProvision, "auto", false, "auto-provision a Jina API key if none available")
	return cmd
}

// serpAutoProvisionJina registers a new Jina key and saves it to the Jina key store.
func serpAutoProvisionJina(jinaStore *jina.KeyStore) (string, error) {
	fmt.Fprintf(os.Stderr, "no keys available — auto-provisioning Jina key...\n")
	acc, err := serp.RegisterProvider(serp.ProviderRegisterOptions{
		Provider: "jina",
		Verbose:  true,
	})
	if err != nil {
		return "", fmt.Errorf("auto-provision failed: %w\nInstall: pip install patchright", err)
	}
	jinaStore.Add(acc.APIKey)
	if err := jinaStore.Save(); err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "provisioned Jina key: %s...%s\n", acc.APIKey[:8], acc.APIKey[len(acc.APIKey)-4:])
	return acc.APIKey, nil
}

// renderSearchResults displays results with lipgloss styling.
func renderSearchResults(query, provider string, result *serp.SearchResult) {
	// Styles
	numStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Bold(true).
		Width(3).
		Align(lipgloss.Right)
	resultTitleStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)
	snippetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Width(80)
	headerStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)
	providerStyle := lipgloss.NewStyle().
		Foreground(warningColor)

	// Header
	fmt.Println()
	fmt.Printf("%s %s  %s\n",
		headerStyle.Render("Search:"),
		titleStyle.Render(query),
		providerStyle.Render(fmt.Sprintf("(%s, %d results)", provider, len(result.OrganicResults))),
	)
	fmt.Println(lipgloss.NewStyle().Foreground(mutedColor).Render(strings.Repeat("─", 80)))
	fmt.Println()

	for i, r := range result.OrganicResults {
		title := fmt.Sprintf("%v", r["title"])
		link := fmt.Sprintf("%v", r["link"])
		snippet := fmt.Sprintf("%v", r["snippet"])

		// Number + Title
		fmt.Printf("%s %s\n", numStyle.Render(fmt.Sprintf("%d.", i+1)), resultTitleStyle.Render(title))
		// URL
		fmt.Printf("    %s\n", urlStyle.Render(link))
		// Snippet
		if snippet != "" && snippet != "<nil>" {
			fmt.Printf("    %s\n", snippetStyle.Render(snippet))
		}
		fmt.Println()
	}
}

// newSerpRotate returns the rotate subcommand.
func newSerpRotate() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate",
		Short: "Check Jina key balances and remove depleted keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			jinaStore, err := jina.LoadKeyStore(jina.DefaultKeyStorePath())
			if err != nil {
				return err
			}

			if len(jinaStore.Keys) == 0 {
				fmt.Println("no Jina keys stored")
				return nil
			}

			var removed int
			for _, k := range jinaStore.Keys {
				info, err := jina.CheckBalance(k.APIKey)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  %s...%s: error: %v\n", k.APIKey[:8], k.APIKey[len(k.APIKey)-4:], err)
					continue
				}

				label := fmt.Sprintf("%s...%s", k.APIKey[:8], k.APIKey[len(k.APIKey)-4:])

				if !info.Valid || info.TotalBalance <= 0 {
					reason := "depleted"
					if !info.Valid {
						reason = "invalid"
					}
					fmt.Printf("  %s: %s (removing)\n", label, errorStyle.Render(reason))
					jinaStore.Remove(k.APIKey)
					removed++
				} else {
					fmt.Printf("  %s: %s\n", label, successStyle.Render(formatTokens(info.TotalBalance)))
					jinaStore.UpdateBalance(k.APIKey, info.TotalBalance)
				}
			}

			if removed > 0 {
				fmt.Printf("removed %d depleted keys\n", removed)
			}

			return jinaStore.Save()
		},
	}
}

func formatTokens(tokens int64) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM tokens", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%dK tokens", tokens/1_000)
	}
	return fmt.Sprintf("%d tokens", tokens)
}

func newSerpList() *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored API keys (Jina keys from ~/data/jina/keys.json)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Jina keys
			jinaStore, err := jina.LoadKeyStore(jina.DefaultKeyStorePath())
			if err != nil {
				return err
			}

			if len(jinaStore.Keys) > 0 {
				fmt.Println(titleStyle.Render("Jina Keys") + "  " + subtitleStyle.Render(jina.DefaultKeyStorePath()))
				fmt.Printf("  %-4s %-24s %-20s %-20s\n",
					labelStyle.Render("#"),
					labelStyle.Render("KEY"),
					labelStyle.Render("BALANCE"),
					labelStyle.Render("CREATED"),
				)

				for i, k := range jinaStore.Keys {
					label := k.APIKey[:8] + "..." + k.APIKey[len(k.APIKey)-4:]

					balanceStr := formatTokens(k.Balance)
					if refresh {
						info, err := jina.CheckBalance(k.APIKey)
						if err != nil {
							balanceStr = errorStyle.Render("error")
						} else if !info.Valid {
							balanceStr = errorStyle.Render("invalid")
						} else {
							jinaStore.UpdateBalance(k.APIKey, info.TotalBalance)
							balanceStr = formatTokens(info.TotalBalance)
						}
					}

					// Color balance
					if k.Balance > 1_000_000 {
						balanceStr = successStyle.Render(balanceStr)
					} else if k.Balance > 0 {
						balanceStr = warningStyle.Render(balanceStr)
					} else if !refresh {
						balanceStr = successStyle.Render(balanceStr) // trust cached
					}

					created := k.CreatedAt.Format("2006-01-02 15:04")
					fmt.Printf("  %-4d %-24s %-20s %-20s\n", i+1, label, balanceStr, created)
				}

				if refresh {
					_ = jinaStore.Save()
				}
				fmt.Println()
			}

			// Legacy serp keys
			store, err := serp.LoadStore(serp.DefaultStorePath())
			if err != nil {
				return err
			}
			if len(store.Accounts) > 0 {
				fmt.Println(titleStyle.Render("Other SERP Keys") + "  " + subtitleStyle.Render(serp.DefaultStorePath()))
				fmt.Printf("  %-12s %-24s %-14s\n",
					labelStyle.Render("PROVIDER"),
					labelStyle.Render("KEY/EMAIL"),
					labelStyle.Render("SEARCHES_LEFT"),
				)
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
					fmt.Printf("  %-12s %-24s %-14d\n", provider, label, a.SearchesLeft)
				}
			}

			if len(jinaStore.Keys) == 0 && len(store.Accounts) == 0 {
				fmt.Println("no keys stored")
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "refresh balances from Jina API")
	return cmd
}

func newSerpInstall() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install dependencies for Jina auto-registration (python3 + patchright)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check python3
			fmt.Print("checking python3... ")
			out, err := exec.Command("python3", "--version").CombinedOutput()
			if err != nil {
				fmt.Println(errorStyle.Render("NOT FOUND"))
				fmt.Println()
				fmt.Println("Install Python 3:")
				fmt.Println("  macOS:  brew install python3")
				fmt.Println("  Ubuntu: apt install python3 python3-pip")
				fmt.Println("  Fedora: dnf install python3 python3-pip")
				return fmt.Errorf("python3 not found")
			}
			fmt.Println(successStyle.Render(strings.TrimSpace(string(out))))

			// Check patchright
			fmt.Print("checking patchright... ")
			out, err = exec.Command("python3", "-c", "import patchright; print(patchright.__version__)").CombinedOutput()
			if err != nil {
				fmt.Println(warningStyle.Render("not installed"))
				fmt.Print("installing patchright... ")
				installArgs := []string{"-m", "pip", "install", "patchright"}
				// Linux may need --break-system-packages
				if runtime.GOOS == "linux" {
					installArgs = append(installArgs, "--break-system-packages")
				}
				out, err = exec.Command("python3", installArgs...).CombinedOutput()
				if err != nil {
					fmt.Println(errorStyle.Render("FAILED"))
					fmt.Printf("  %s\n", string(out))
					fmt.Println()
					fmt.Println("Manual install:")
					fmt.Println("  pip install patchright")
					fmt.Println("  # or on Ubuntu 24+:")
					fmt.Println("  pip install --break-system-packages patchright")
					return fmt.Errorf("patchright install failed")
				}
				fmt.Println(successStyle.Render("OK"))
			} else {
				fmt.Println(successStyle.Render(strings.TrimSpace(string(out))))
			}

			// Install patchright browser
			fmt.Print("checking patchright browser... ")
			out, err = exec.Command("python3", "-c",
				"from patchright.sync_api import sync_playwright; p=sync_playwright().start(); b=p.chromium.launch(headless=True); b.close(); p.stop(); print('OK')").CombinedOutput()
			if err != nil {
				fmt.Println(warningStyle.Render("not installed"))
				fmt.Print("installing chromium browser... ")
				out, err = exec.Command("python3", "-m", "patchright", "install", "chromium").CombinedOutput()
				if err != nil {
					fmt.Println(errorStyle.Render("FAILED"))
					fmt.Printf("  %s\n", string(out))
					return fmt.Errorf("browser install failed")
				}
				fmt.Println(successStyle.Render("OK"))

				// Install system deps on Linux
				if runtime.GOOS == "linux" {
					fmt.Print("installing system dependencies... ")
					out, err = exec.Command("python3", "-m", "patchright", "install-deps", "chromium").CombinedOutput()
					if err != nil {
						fmt.Println(warningStyle.Render("may need sudo"))
						fmt.Println("  Run manually: python3 -m patchright install-deps chromium")
					} else {
						fmt.Println(successStyle.Render("OK"))
					}
				}
			} else {
				fmt.Println(successStyle.Render("OK"))
			}

			// Check xvfb on Linux (needed for Turnstile — headless can't solve it)
			if runtime.GOOS == "linux" {
				fmt.Print("checking xvfb... ")
				_, err = exec.LookPath("xvfb-run")
				if err != nil {
					fmt.Println(warningStyle.Render("not installed"))
					fmt.Print("installing xvfb... ")
					out, err = exec.Command("apt-get", "install", "-y", "xvfb").CombinedOutput()
					if err != nil {
						fmt.Println(warningStyle.Render("may need sudo"))
						fmt.Println("  Run manually: apt install xvfb")
					} else {
						fmt.Println(successStyle.Render("OK"))
					}
				} else {
					fmt.Println(successStyle.Render("OK"))
				}
			}

			fmt.Println()
			fmt.Println(successStyle.Render("All dependencies installed!"))
			fmt.Println("  Run: search serp signup jina --verbose")
			fmt.Println("  Or:  search serp search \"your query\" --auto")
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

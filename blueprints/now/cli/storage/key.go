package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "key",
		Short:   "Manage API keys",
		Aliases: []string{"keys"},
	}

	cmd.AddCommand(
		newKeyCreateCmd(),
		newKeyListCmd(),
		newKeyRmCmd(),
	)
	return cmd
}

func newKeyCreateCmd() *cobra.Command {
	var (
		prefix  string
		expires string
	)

	cmd := &cobra.Command{
		Use:     "create <name>",
		Short:   "Create an API key",
		Aliases: []string{"new"},
		Example: `  storage key create deploy
  storage key create ci --prefix deploy/
  storage key create bot --expires 30d`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			name := args[0]
			body := map[string]any{"name": name}

			if prefix != "" {
				body["prefix"] = prefix
			}
			if expires != "" {
				body["expires_in"] = parseDuration(expires)
			}

			data, err := d.Client.DoJSON("POST", "/auth/keys", body)
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			var resp struct {
				Token string `json:"token"`
				ID    string `json:"id"`
			}
			json.Unmarshal(data, &resp)

			fmt.Println()
			fmt.Printf("%s %s\n", d.Out.bold("API Key:"), resp.Token)
			fmt.Println()
			fmt.Println(d.Out.dim("Save this key. It won't be shown again."))
			fmt.Printf("Use it with: %s\n", d.Out.cyan("STORAGE_TOKEN="+resp.Token+" storage ls"))
			return nil
		}),
	}

	cmd.Flags().StringVarP(&prefix, "prefix", "p", "", "path prefix restriction")
	cmd.Flags().StringVarP(&expires, "expires", "x", "", "expiration (7d, 30d, 90d)")
	return cmd
}

func newKeyListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List API keys",
		Aliases: []string{"ls"},
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			data, err := d.Client.Get("/auth/keys")
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			var resp struct {
				Keys []struct {
					ID        string `json:"id"`
					Name      string `json:"name"`
					Prefix    string `json:"prefix"`
					CreatedAt int64  `json:"created_at"`
				} `json:"keys"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return err
			}

			if len(resp.Keys) == 0 {
				d.Out.Info("No API keys", "Create one with: storage key create <name>")
				return nil
			}

			fmt.Printf("%-24s  %-20s  %-16s  %s\n", "ID", "NAME", "PREFIX", "CREATED")
			for _, k := range resp.Keys {
				prefix := "*"
				if k.Prefix != "" {
					prefix = k.Prefix
				}
				fmt.Printf("%-24s  %-20s  %-16s  %s\n", k.ID, k.Name, prefix, RelativeTime(k.CreatedAt))
			}
			return nil
		}),
	}
}

func newKeyRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <id>",
		Short:   "Revoke an API key",
		Aliases: []string{"revoke", "delete"},
		Args:    cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			id := strings.TrimSpace(args[0])
			if _, err := d.Client.Delete("/auth/keys/" + id); err != nil {
				return err
			}

			d.Out.Info("Revoked", "API key "+id)
			fmt.Fprintln(os.Stderr)
			return nil
		}),
	}
}

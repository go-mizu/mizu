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
		newKeyRevokeCmd(),
	)
	return cmd
}

func newKeyCreateCmd() *cobra.Command {
	var (
		scope  string
		prefix string
	)

	cmd := &cobra.Command{
		Use:     "create <name>",
		Short:   "Create an API key",
		Aliases: []string{"new"},
		Example: `  storage key create ci-deploy
  storage key create ci --scope "object:read,object:write"
  storage key create readonly --scope "object:read" --prefix docs/`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			name := args[0]
			body := map[string]any{"name": name}

			if scope != "" {
				scopes := strings.Split(scope, ",")
				for i, s := range scopes {
					scopes[i] = strings.TrimSpace(s)
				}
				body["scopes"] = scopes
			}
			if prefix != "" {
				body["path_prefix"] = prefix
			}

			data, err := d.Client.DoJSON("POST", "/keys", body)
			if err != nil {
				return err
			}

			if globalFlags.json {
				var raw json.RawMessage
				json.Unmarshal(data, &raw)
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(raw)
			}

			var resp struct {
				Key string `json:"key"`
			}
			json.Unmarshal(data, &resp)

			fmt.Println()
			fmt.Printf("%s %s\n", d.Out.bold("API Key:"), resp.Key)
			fmt.Println()
			fmt.Println(d.Out.dim("Save this key — it won't be shown again."))
			fmt.Printf("Use it with: %s\n", d.Out.cyan("export STORAGE_TOKEN="+resp.Key))
			return nil
		}),
	}

	cmd.Flags().StringVarP(&scope, "scope", "s", "", "scopes (comma-separated)")
	cmd.Flags().StringVarP(&prefix, "prefix", "p", "", "path prefix restriction")
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

			data, err := d.Client.Get("/keys")
			if err != nil {
				return err
			}

			if globalFlags.json {
				var raw json.RawMessage
				json.Unmarshal(data, &raw)
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(raw)
			}

			var resp struct {
				Items []struct {
					ID        string   `json:"id"`
					Name      string   `json:"name"`
					Scopes    []string `json:"scopes"`
					CreatedAt int64    `json:"created_at"`
				} `json:"items"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return err
			}
			keys := resp.Items

			if len(keys) == 0 {
				d.Out.Info("No API keys", "Create one with: storage key create <name>")
				return nil
			}

			fmt.Printf("%-8s %-20s %-20s  %s\n", "ID", "NAME", "SCOPES", "CREATED")
			for _, k := range keys {
				scopes := "*"
				if len(k.Scopes) > 0 {
					scopes = strings.Join(k.Scopes, ",")
				}
				fmt.Printf("%-8s %-20s %-20s  %s\n", k.ID, k.Name, scopes, RelativeTime(k.CreatedAt))
			}
			return nil
		}),
	}
}

func newKeyRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "revoke <id>",
		Short:   "Revoke an API key",
		Aliases: []string{"rm", "delete"},
		Args:    cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			id := args[0]
			if _, err := d.Client.Delete("/keys/" + id); err != nil {
				return err
			}

			d.Out.Info("Revoked", "API key "+id)
			return nil
		}),
	}
}

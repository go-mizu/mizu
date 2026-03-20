package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newFindCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "find <query>",
		Short:   "Search files by name",
		Aliases: []string{"search"},
		Example: `  storage find quarterly
  storage find "*.pdf" --json`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			query := strings.TrimSpace(args[0])
			if query == "" {
				return &CLIError{Code: ExitUsage, Msg: "query required", Hint: "Usage: storage find <query>"}
			}

			params := url.Values{}
			params.Set("q", query)
			if limit > 0 {
				params.Set("limit", strconv.Itoa(limit))
			}

			data, err := d.Client.Get("/find?" + params.Encode())
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			var resp struct {
				Results []struct {
					Name string `json:"name"`
					Path string `json:"path"`
				} `json:"results"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return err
			}

			if len(resp.Results) == 0 {
				d.Out.Info("No results", "for '"+query+"'")
				return nil
			}

			fmt.Printf("%-40s  %s\n", "PATH", "NAME")
			for _, r := range resp.Results {
				fmt.Printf("%-40s  %s\n", r.Path, r.Name)
			}
			return nil
		}),
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "max results")
	return cmd
}

package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newLsCmd() *cobra.Command {
	var (
		limit  int
		offset int
	)

	cmd := &cobra.Command{
		Use:     "ls [path]",
		Short:   "List files and directories",
		Aliases: []string{"list"},
		Example: `  storage ls
  storage ls docs/
  storage ls docs/ --json`,
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			prefix := ""
			if len(args) > 0 {
				prefix = strings.TrimPrefix(args[0], "/")
			}

			params := url.Values{}
			if limit > 0 {
				params.Set("limit", strconv.Itoa(limit))
			}
			if offset > 0 {
				params.Set("offset", strconv.Itoa(offset))
			}

			if prefix != "" {
				params.Set("prefix", prefix)
			}
			apiPath := "/files"
			if len(params) > 0 {
				apiPath += "?" + params.Encode()
			}

			data, err := d.Client.Get(apiPath)
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			var resp struct {
				Entries []struct {
					Name      string `json:"name"`
					Type      string `json:"type"`
					Size      int64  `json:"size"`
					UpdatedAt int64  `json:"updated_at"`
				} `json:"entries"`
				Truncated bool `json:"truncated"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return err
			}

			if len(resp.Entries) == 0 {
				hint := "Upload a file with: storage put <file> " + prefix
				if prefix == "" {
					hint = "Upload a file with: storage put <file>"
				}
				d.Out.Info("Empty", hint)
				return nil
			}

			fmt.Printf("%-32s %10s  %-12s  %s\n", "NAME", "SIZE", "TYPE", "MODIFIED")
			for _, e := range resp.Entries {
				size := HumanSize(e.Size)
				display := "file"
				if e.Type == "directory" {
					size = "-"
					display = d.Out.cyan("directory/")
				}
				fmt.Printf("%-32s %10s  %-12s  %s\n",
					e.Name, size, display, RelativeTime(e.UpdatedAt))
			}

			if resp.Truncated {
				fmt.Fprintf(os.Stderr, "\n%s\n", d.Out.dim("More results available. Use --limit and --offset to paginate."))
			}
			return nil
		}),
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "skip N results")
	return cmd
}

// printJSON pretty-prints raw JSON to stdout.
func printJSON(data []byte) error {
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		os.Stdout.Write(data)
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(raw)
}

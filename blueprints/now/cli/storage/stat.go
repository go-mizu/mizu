package storage

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newStatCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "stat",
		Short:   "Show storage usage",
		Aliases: []string{"stats"},
		Example: `  storage stat
  storage stat --json`,
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			data, err := d.Client.Get("/files/stats")
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			var stats struct {
				Files int   `json:"files"`
				Bytes int64 `json:"bytes"`
			}
			if err := json.Unmarshal(data, &stats); err != nil {
				return err
			}

			fmt.Printf("%-14s %d\n", d.Out.dim("Files:"), stats.Files)
			fmt.Printf("%-14s %s\n", d.Out.dim("Total size:"), HumanSize(stats.Bytes))
			return nil
		}),
	}
}

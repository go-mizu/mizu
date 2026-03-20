package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show storage usage",
		Example: `  storage stats
  storage stats --json`,
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			data, err := d.Client.Get("/drive/stats")
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

			var stats struct {
				FileCount   int   `json:"file_count"`
				FolderCount int   `json:"folder_count"`
				TotalSize   int64 `json:"total_size"`
				TrashCount  int   `json:"trash_count"`
				Quota       int64 `json:"quota"`
			}
			if err := json.Unmarshal(data, &stats); err != nil {
				return err
			}

			if stats.Quota == 0 {
				stats.Quota = 5 * 1024 * 1024 * 1024 // 5 GB default
			}

			pct := float64(0)
			if stats.Quota > 0 {
				pct = float64(stats.TotalSize) * 100 / float64(stats.Quota)
			}

			w := func(label, value string) {
				fmt.Printf("%-14s %s\n", d.Out.dim(label), value)
			}
			w("Files:", fmt.Sprintf("%d", stats.FileCount))
			w("Folders:", fmt.Sprintf("%d", stats.FolderCount))
			w("Total size:", HumanSize(stats.TotalSize))
			w("Quota:", fmt.Sprintf("%s (%.1f%% used)", HumanSize(stats.Quota), pct))
			return nil
		}),
	}
}

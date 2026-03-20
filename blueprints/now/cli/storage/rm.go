package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:     "rm <bucket/path...>",
		Short:   "Delete objects",
		Aliases: []string{"delete", "del"},
		Example: `  storage rm docs/old-report.pdf
  storage rm docs/draft1.md docs/draft2.md
  storage rm docs/drafts/ --recursive`,
		Args: cobra.MinimumNArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			// All paths must be in the same bucket
			var bucket string
			var paths []string

			for _, p := range args {
				if !strings.Contains(p, "/") {
					return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage rm docs/file.txt"}
				}
				b, objPath, _ := strings.Cut(p, "/")
				if bucket == "" {
					bucket = b
				} else if bucket != b {
					return &CLIError{Code: ExitUsage, Msg: "mixed buckets", Hint: "All paths must be in the same bucket for batch delete"}
				}
				paths = append(paths, objPath)
			}

			// Recursive delete
			if recursive && len(paths) == 1 {
				prefix := paths[0]

				listBody := map[string]any{
					"prefix": prefix,
					"limit":  1000,
				}
				data, err := d.Client.DoJSON("POST", "/object/list/"+bucket, listBody)
				if err != nil {
					return err
				}

				var objects []objectInfo
				if err := json.Unmarshal(data, &objects); err != nil {
					return err
				}

				if len(objects) == 0 {
					d.Out.Info("Nothing", "to delete")
					return nil
				}

				// Confirm on TTY
				if d.Out.IsTTY && !globalFlags.quiet {
					fmt.Fprintf(os.Stderr, "This will delete %d objects. Continue? [y/N] ", len(objects))
					scanner := bufio.NewScanner(os.Stdin)
					if scanner.Scan() {
						answer := strings.TrimSpace(scanner.Text())
						if !strings.HasPrefix(strings.ToLower(answer), "y") {
							return nil
						}
					}
				}

				delPaths := make([]string, len(objects))
				for i, o := range objects {
					delPaths[i] = o.Path
				}

				delBody := map[string]any{"paths": delPaths}
				if _, err := d.Client.DoJSON("DELETE", "/object/"+bucket, delBody); err != nil {
					return err
				}

				d.Out.Info("Deleted", fmt.Sprintf("%d objects from %s", len(objects), bucket))
				return nil
			}

			// Batch delete
			delBody := map[string]any{"paths": paths}
			if _, err := d.Client.DoJSON("DELETE", "/object/"+bucket, delBody); err != nil {
				return err
			}

			if len(paths) == 1 {
				d.Out.Info("Deleted", bucket+"/"+paths[0])
			} else {
				d.Out.Info("Deleted", fmt.Sprintf("%d objects from %s", len(paths), bucket))
			}
			return nil
		}),
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "delete all objects with prefix")
	return cmd
}

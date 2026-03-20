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
	var (
		recursive bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:     "rm <path...>",
		Short:   "Delete files or directories",
		Aliases: []string{"delete", "del"},
		Example: `  storage rm docs/old-report.pdf
  storage rm logs/ --recursive
  storage rm logs/ --recursive --force`,
		Args: cobra.MinimumNArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			for _, path := range args {
				path = strings.TrimPrefix(path, "/")

				// If recursive and path doesn't end with /, add it
				if recursive && !strings.HasSuffix(path, "/") {
					path += "/"
				}

				isDir := strings.HasSuffix(path, "/")

				// Confirm directory deletion on TTY
				if isDir && d.Out.IsTTY && !globalFlags.quiet && !force {
					fmt.Fprintf(os.Stderr, "Delete everything under %s? [y/N] ", path)
					scanner := bufio.NewScanner(os.Stdin)
					if scanner.Scan() {
						answer := strings.TrimSpace(scanner.Text())
						if !strings.HasPrefix(strings.ToLower(answer), "y") {
							continue
						}
					}
				}

				data, err := d.Client.Delete("/f/" + path)
				if err != nil {
					return err
				}

				if globalFlags.json {
					printJSON(data)
					continue
				}

				var resp struct {
					Deleted int `json:"deleted"`
				}
				json.Unmarshal(data, &resp)

				if isDir {
					d.Out.Info("Deleted", fmt.Sprintf("%s (%d files)", path, resp.Deleted))
				} else {
					d.Out.Info("Deleted", path)
				}
			}
			return nil
		}),
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "delete directory and contents")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	return cmd
}

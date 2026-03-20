package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var public bool

	cmd := &cobra.Command{
		Use:     "get <bucket/path> [local-path]",
		Short:   "Download a file (or stdout with -)",
		Aliases: []string{"download", "pull"},
		Example: `  storage get docs/report.pdf
  storage get docs/report.pdf ~/Desktop/report.pdf
  storage get docs/data.csv - | wc -c
  storage get --public avatars/logo.png`,
		Args: cobra.RangeArgs(1, 2),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if !public {
				if err := RequireToken(d.Config); err != nil {
					return err
				}
			}

			src := args[0]
			if !strings.Contains(src, "/") {
				return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage get docs/file.txt"}
			}

			bucket, objPath, _ := strings.Cut(src, "/")

			dest := ""
			if len(args) > 1 {
				dest = args[1]
			}
			if dest == "" {
				dest = filepath.Base(objPath)
			}

			urlPath := fmt.Sprintf("/object/%s/%s", bucket, objPath)
			if public {
				urlPath = fmt.Sprintf("/object/public/%s/%s", bucket, objPath)
			}

			if dest == "-" {
				return d.Client.Download(urlPath, os.Stdout)
			}

			// Ensure parent dir exists
			dir := filepath.Dir(dest)
			if dir != "." {
				os.MkdirAll(dir, 0o755)
			}

			f, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer f.Close()

			if err := d.Client.Download(urlPath, f); err != nil {
				os.Remove(dest)
				return err
			}

			if !globalFlags.quiet {
				fi, _ := f.Stat()
				sizeStr := ""
				if fi != nil {
					sizeStr = " (" + HumanSize(fi.Size()) + ")"
				}
				d.Out.Info("Downloaded", filepath.Base(objPath)+sizeStr)
			}
			return nil
		}),
	}

	cmd.Flags().BoolVarP(&public, "public", "p", false, "download from public bucket (no auth)")
	return cmd
}

func newCatCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "cat <bucket/path>",
		Short:   "Print file contents to stdout",
		Aliases: []string{},
		Example: `  storage cat docs/readme.md
  storage cat docs/config.json | jq .database`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			src := args[0]
			if !strings.Contains(src, "/") {
				return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage cat docs/file.txt"}
			}

			bucket, objPath, _ := strings.Cut(src, "/")
			urlPath := fmt.Sprintf("/object/%s/%s", bucket, objPath)
			return d.Client.Download(urlPath, os.Stdout)
		}),
	}
}

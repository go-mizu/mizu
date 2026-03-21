package storage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <path> [dest]",
		Short:   "Download a file (or stdout with -)",
		Aliases: []string{"download", "pull"},
		Example: `  storage get docs/report.pdf
  storage get docs/report.pdf ~/Downloads/
  storage get docs/data.csv -`,
		Args: cobra.RangeArgs(1, 2),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			src := strings.TrimPrefix(args[0], "/")

			dest := ""
			if len(args) > 1 {
				dest = args[1]
			}
			if dest == "" {
				dest = filepath.Base(src)
			}

			apiPath := "/files/" + src

			// Write to stdout
			if dest == "-" {
				return d.Client.Download(apiPath, os.Stdout)
			}

			// If dest is an existing directory, append filename
			if info, err := os.Stat(dest); err == nil && info.IsDir() {
				dest = filepath.Join(dest, filepath.Base(src))
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

			if err := d.Client.Download(apiPath, f); err != nil {
				os.Remove(dest)
				return err
			}

			fi, _ := f.Stat()
			sizeStr := ""
			if fi != nil {
				sizeStr = " (" + HumanSize(fi.Size()) + ")"
			}
			d.Out.Info("Downloaded", filepath.Base(src)+sizeStr)
			return nil
		}),
	}
	return cmd
}

func newCatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat <path>",
		Short: "Print file contents to stdout",
		Example: `  storage cat docs/config.json
  storage cat docs/data.json | jq .`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			src := strings.TrimPrefix(args[0], "/")
			return d.Client.Download("/files/"+src, os.Stdout)
		}),
	}
}

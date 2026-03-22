package storage

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newPutCmd() *cobra.Command {
	var contentType string

	cmd := &cobra.Command{
		Use:     "put <file> [path]",
		Short:   "Upload a file (or stdin with -)",
		Aliases: []string{"upload", "push"},
		Example: `  storage put report.pdf docs/
  storage put photo.jpg images/photo.jpg
  echo "hello" | storage put - notes/hello.txt`,
		Args: cobra.RangeArgs(1, 2),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			file := args[0]
			dest := ""
			if len(args) > 1 {
				dest = args[1]
			}

			// Resolve destination path
			dest = strings.TrimPrefix(dest, "/")

			if dest == "" {
				if file == "-" {
					return &CLIError{
						Code: ExitUsage,
						Msg:  "path required when reading from stdin",
						Hint: "Usage: echo data | storage put - path/filename.txt",
					}
				}
				dest = filepath.Base(file)
			} else if strings.HasSuffix(dest, "/") {
				if file == "-" {
					return &CLIError{
						Code: ExitUsage,
						Msg:  "cannot use trailing slash with stdin",
						Hint: "Provide a full path: storage put - path/filename.txt",
					}
				}
				dest += filepath.Base(file)
			}

			// Determine content type
			ct := contentType
			if ct == "" && file != "-" {
				ct = DetectContentType(file)
			}
			if ct == "" {
				ct = "application/octet-stream"
			}

			// Read source into memory for SHA-256 computation
			var fileData []byte
			if file == "-" {
				var err error
				fileData, err = io.ReadAll(os.Stdin)
				if err != nil {
					return &CLIError{Code: ExitError, Msg: "failed to read stdin"}
				}
			} else {
				var err error
				fileData, err = os.ReadFile(file)
				if err != nil {
					if os.IsNotExist(err) {
						return &CLIError{Code: ExitNotFound, Msg: "file not found: " + file}
					}
					return err
				}
			}

			result, err := d.Client.UploadPresigned(dest, fileData, ct)
			if err != nil {
				return err
			}

			if globalFlags.json {
				jsonData, _ := json.Marshal(result)
				return printJSON(jsonData)
			}

			sizeStr := ""
			if result.Size > 0 {
				sizeStr = " (" + HumanSize(result.Size) + ")"
			}
			displayPath := result.Path
			if displayPath == "" {
				displayPath = dest
			}
			action := "Uploaded"
			if result.Deduplicated {
				action = "Deduplicated"
			}
			d.Out.Info(action, displayPath+sizeStr)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&contentType, "type", "T", "", "content type (auto-detected)")
	return cmd
}

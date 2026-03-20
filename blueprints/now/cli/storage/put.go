package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newPutCmd() *cobra.Command {
	var (
		contentType string
		bucketFlag  string
	)

	cmd := &cobra.Command{
		Use:     "put <file> [bucket[/path]]",
		Short:   "Upload a file (or stdin with -)",
		Aliases: []string{"upload", "push"},
		Example: `  storage put report.pdf docs
  storage put photo.jpg pics/vacation/
  echo "hello" | storage put - docs/hello.txt
  storage put image.png avatars --type image/png`,
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

			// Resolve bucket and path
			bucket := bucketFlag
			objPath := ""

			if dest != "" {
				if strings.Contains(dest, "/") {
					parts := strings.SplitN(dest, "/", 2)
					if bucket == "" {
						bucket = parts[0]
					}
					objPath = parts[1]
				} else {
					if bucket == "" {
						bucket = dest
					}
				}
			}

			if bucket == "" {
				bucket = d.Config.Bucket
			}
			if bucket == "" {
				bucket = "default"
			}

			if objPath == "" {
				if file == "-" {
					return &CLIError{
						Code: ExitUsage,
						Msg:  "path required",
						Hint: "When reading from stdin, specify a destination path\nUsage: echo data | storage put - bucket/filename.txt",
					}
				}
				objPath = filepath.Base(file)
			}

			// Determine content type
			ct := contentType
			if ct == "" && file != "-" {
				ct = DetectContentType(file)
			}
			if ct == "" {
				ct = "application/octet-stream"
			}

			// Open file or stdin
			var r *os.File
			if file == "-" {
				r = os.Stdin
			} else {
				var err error
				r, err = os.Open(file)
				if err != nil {
					if os.IsNotExist(err) {
						return &CLIError{Code: ExitNotFound, Msg: "file not found", Hint: file + " does not exist"}
					}
					return err
				}
				defer r.Close()
			}

			apiPath := fmt.Sprintf("/object/%s/%s", bucket, objPath)
			data, err := d.Client.Upload("PUT", apiPath, r, ct)
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
				Size int64 `json:"size"`
			}
			json.Unmarshal(data, &resp)

			sizeStr := ""
			if resp.Size > 0 {
				sizeStr = " (" + HumanSize(resp.Size) + ")"
			}
			d.Out.Info("Uploaded", objPath+sizeStr+" to "+bucket)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&contentType, "type", "T", "", "content type (auto-detected)")
	cmd.Flags().StringVarP(&bucketFlag, "bucket", "b", "", "target bucket")
	return cmd
}

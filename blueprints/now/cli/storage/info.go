package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <bucket/path>",
		Short: "Show object metadata",
		Example: `  storage info docs/report.pdf`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			src := args[0]
			if !strings.Contains(src, "/") {
				return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage info docs/file.txt"}
			}

			bucket, objPath, _ := strings.Cut(src, "/")

			data, err := d.Client.Get(fmt.Sprintf("/object/info/%s/%s", bucket, objPath))
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

			var obj struct {
				Name        string `json:"name"`
				Path        string `json:"path"`
				Size        int64  `json:"size"`
				ContentType string `json:"content_type"`
				CreatedAt   int64  `json:"created_at"`
				UpdatedAt   int64  `json:"updated_at"`
			}
			if err := json.Unmarshal(data, &obj); err != nil {
				return err
			}

			w := func(label, value string) {
				fmt.Printf("%-12s %s\n", d.Out.dim(label), value)
			}
			w("Path:", objPath)
			w("Bucket:", bucket)
			w("Size:", HumanSize(obj.Size))
			w("Type:", obj.ContentType)
			w("Created:", RelativeTime(obj.CreatedAt))
			w("Modified:", RelativeTime(obj.UpdatedAt))
			return nil
		}),
	}
}

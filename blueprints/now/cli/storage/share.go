package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newShareCmd() *cobra.Command {
	var expires string

	cmd := &cobra.Command{
		Use:     "share <bucket/path>",
		Short:   "Create a signed download URL",
		Aliases: []string{"sign"},
		Example: `  storage share docs/report.pdf
  storage share docs/report.pdf --expires 7d
  storage share docs/report.pdf | pbcopy`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			src := args[0]
			if !strings.Contains(src, "/") {
				return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage share docs/file.txt"}
			}

			bucket, objPath, _ := strings.Cut(src, "/")
			expiresSec := parseDuration(expires)

			body := map[string]any{
				"path":       objPath,
				"expires_in": expiresSec,
			}

			data, err := d.Client.DoJSON("POST", "/object/sign/"+bucket, body)
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
				SignedURL string `json:"signed_url"`
			}
			json.Unmarshal(data, &resp)

			fmt.Println(d.Config.Endpoint + resp.SignedURL)
			d.Out.Info("Expires", "in "+expires)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&expires, "expires", "x", "1h", "expiration duration (30m, 1h, 1d, 7d)")
	return cmd
}

// parseDuration converts a duration string like "30m", "1h", "7d" to seconds.
func parseDuration(s string) int {
	if s == "" {
		return 3600
	}

	n := len(s)
	if n < 2 {
		v, _ := strconv.Atoi(s)
		return v
	}

	numStr := s[:n-1]
	unit := s[n-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 3600
	}

	switch unit {
	case 's':
		return num
	case 'm':
		return num * 60
	case 'h':
		return num * 3600
	case 'd':
		return num * 86400
	default:
		v, _ := strconv.Atoi(s)
		return v
	}
}

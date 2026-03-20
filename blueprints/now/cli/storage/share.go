package storage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newShareCmd() *cobra.Command {
	var expires string

	cmd := &cobra.Command{
		Use:   "share <path>",
		Short: "Create a temporary share link",
		Example: `  storage share docs/report.pdf
  storage share docs/report.pdf --expires 7d
  storage share pic.jpg --json`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			path := strings.TrimPrefix(args[0], "/")
			ttl := parseDuration(expires)

			body := map[string]any{
				"path": path,
				"ttl":  ttl,
			}

			data, err := d.Client.DoJSON("POST", "/share", body)
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			var resp struct {
				URL       string `json:"url"`
				ExpiresAt int64  `json:"expires_at"`
			}
			json.Unmarshal(data, &resp)

			fmt.Println(resp.URL)
			d.Out.Info("Expires", "in "+expires)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&expires, "expires", "x", "1h", "expiration (30m, 1h, 7d)")
	return cmd
}

// parseDuration converts "30m", "1h", "7d" to seconds.
func parseDuration(s string) int {
	if s == "" {
		return 3600
	}

	n := len(s)
	if n < 2 {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 3600
		}
		return v
	}

	numStr := s[:n-1]
	unit := s[n-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		// Try parsing the whole string as seconds
		v, err2 := strconv.Atoi(s)
		if err2 != nil {
			return 3600
		}
		return v
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

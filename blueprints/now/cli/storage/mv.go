package storage

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "mv <bucket/from> <bucket/to>",
		Short:   "Move or rename an object",
		Aliases: []string{"move", "rename"},
		Example: `  storage mv docs/old-name.pdf docs/new-name.pdf
  storage mv docs/report.pdf docs/archive/report.pdf`,
		Args: cobra.ExactArgs(2),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			from, to := args[0], args[1]

			if !strings.Contains(from, "/") || !strings.Contains(to, "/") {
				return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage mv docs/old.md docs/new.md"}
			}

			bucketFrom, pathFrom, _ := strings.Cut(from, "/")
			bucketTo, pathTo, _ := strings.Cut(to, "/")

			if bucketFrom != bucketTo {
				return &CLIError{
					Code: ExitUsage,
					Msg:  "cross-bucket move",
					Hint: "Both paths must be in the same bucket\nUse 'storage cp' + 'storage rm' for cross-bucket moves",
				}
			}

			body := map[string]string{
				"bucket": bucketFrom,
				"from":   pathFrom,
				"to":     pathTo,
			}

			data, err := d.Client.DoJSON("POST", "/object/move", body)
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

			d.Out.Info("Moved", pathFrom+" -> "+pathTo)
			return nil
		}),
	}
}

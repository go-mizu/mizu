package storage

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newCpCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "cp <bucket/from> <bucket/to>",
		Short:   "Copy an object",
		Aliases: []string{"copy"},
		Example: `  storage cp docs/template.md docs/new-doc.md`,
		Args:    cobra.ExactArgs(2),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			from, to := args[0], args[1]

			if !strings.Contains(from, "/") || !strings.Contains(to, "/") {
				return &CLIError{Code: ExitUsage, Msg: "invalid path", Hint: "Use bucket/path format: storage cp docs/a.md docs/b.md"}
			}

			bucketFrom, pathFrom, _ := strings.Cut(from, "/")
			bucketTo, pathTo, _ := strings.Cut(to, "/")

			body := map[string]string{
				"from_bucket": bucketFrom,
				"from_path":   pathFrom,
				"to_bucket":   bucketTo,
				"to_path":     pathTo,
			}

			data, err := d.Client.DoJSON("POST", "/object/copy", body)
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

			d.Out.Info("Copied", pathFrom+" -> "+pathTo)
			return nil
		}),
	}
}

package storage

import (
	"strings"

	"github.com/spf13/cobra"
)

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "mv <from> <to>",
		Short:   "Move or rename a file",
		Aliases: []string{"move", "rename"},
		Example: `  storage mv draft.md published/post.md
  storage mv old/report.pdf archive/report.pdf`,
		Args: cobra.ExactArgs(2),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			from := strings.TrimPrefix(args[0], "/")
			to := strings.TrimPrefix(args[1], "/")

			body := map[string]string{
				"from": from,
				"to":   to,
			}

			data, err := d.Client.DoJSON("POST", "/files/move", body)
			if err != nil {
				return err
			}

			if globalFlags.json {
				return printJSON(data)
			}

			d.Out.Info("Moved", from+" -> "+to)
			return nil
		}),
	}
}

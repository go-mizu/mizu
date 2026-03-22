package chat

import (
	"encoding/json"
	"os"

	pkgchat "now/pkg/chat"

	"github.com/spf13/cobra"
)

func newGetCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ar, err := signAndVerify(cmd, d, "get", map[string]string{
				"id": args[0],
			})
			if err != nil {
				return err
			}

			c, err := d.svc.Get(cmd.Context(), pkgchat.GetInput{
				ID: args[0],
			}, ar.Actor)
			if err != nil {
				return err
			}

			return json.NewEncoder(os.Stdout).Encode(c)
		},
	}
}
